//go:build extism

package main

import (
	"context"
	"encoding/json"
	"example.com/support-shell/internal/core"
	"example.com/support-shell/pkg/pluginapi"
	"example.com/support-shell/internal/pluginmanager"
	"fmt"
	extism "github.com/extism/go-sdk"
	"os"
	"strings"
	"path/filepath"
	"sync"
	"sort"
	"strconv"
	"time"
)


func loadPlugins(r *core.Registry, manager *pluginmanager.Manager) error {
    // PLUGIN_PATH is a directory, a WASM file, or a comma-separated list.
    source := strings.TrimSpace(os.Getenv("PLUGIN_PATH"))
    if source == "" {
        source = strings.TrimSpace(os.Getenv("SUPPORT_PLUGINS"))
    }
    if source == "" {
        source = strings.TrimSpace(os.Getenv("SUPPORT_PLUGIN"))
    }
    if source == "" { source = "./plugins" }

    seenPaths, seenNames := map[string]bool{}, map[string]bool{}
    var files []string
    for _, item := range strings.Split(source, ",") {
        item = strings.TrimSpace(item)
        if item == "" { return fmt.Errorf("empty plugin path") }
        info, err := os.Stat(item)
        if os.IsNotExist(err) && source == "./plugins" { continue } // no plugins installed yet
        if err != nil { return fmt.Errorf("plugin path %s: %w", item, err) }
        candidates := []string{item}
        if info.IsDir() {
            candidates = nil
            for _, pattern := range []string{"*.wasm", "*/target/wasm32-wasip1/release/*.wasm"} {
                hits, err := filepath.Glob(filepath.Join(item, pattern))
                if err != nil { return err }
                candidates = append(candidates, hits...)
            }
        } else if filepath.Ext(item) != ".wasm" {
            return fmt.Errorf("plugin must be .wasm: %s", item)
        }
        for _, candidate := range candidates {
            abs, err := filepath.Abs(candidate)
            if err != nil { return err }
            if seenPaths[abs] { return fmt.Errorf("duplicate WASM plugin: %s", candidate) }
            seenPaths[abs] = true
            files = append(files, abs)
        }
    }
    sort.Strings(files)
    for _, path := range files {
        if err := loadSinglePlugin(r, manager, path, seenNames); err != nil {
            return fmt.Errorf("load WASM plugin %s: %w", path, err)
        }
    }
    return nil
}

func loadSinglePlugin(r *core.Registry, manager *pluginmanager.Manager, path string, seenNames map[string]bool) error {
	// Each plugin has an independent, manifest-scoped capability set.
	// The host function is restricted to read-only built-in commands.
	var granted map[string]bool
	var callMu sync.Mutex
	host := extism.NewHostFunctionWithStack("host_call", func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		raw, err := p.ReadBytes(stack[0])
		if err != nil {
			return
		}
		var req struct {
			Command string         `json:"command"`
			Input   map[string]any `json:"input"`
		}
		if err = json.Unmarshal(raw, &req); err != nil {
			return
		}
		if !granted[req.Command] || !allowedPluginCall(req.Command) {
			return
		}
		result, err := r.Execute(ctx, req.Command, req.Input)
		var out []byte
		if err != nil {
			out, _ = json.Marshal(map[string]string{"error": err.Error()})
		} else {
			out, _ = json.Marshal(result)
		}
		stack[0], _ = p.WriteBytes(out)
	}, []extism.ValueType{extism.ValueTypePTR}, []extism.ValueType{extism.ValueTypePTR})
	timeout := pluginCallTimeout()
    // Extism go-sdk v1.7.1 enables wazero WithCloseOnContextDone only
    // when manifest.Timeout is non-zero. An external Go context alone
    // cannot interrupt a guest in an infinite loop.
    manifest := extism.Manifest{Wasm: []extism.Wasm{extism.WasmFile{Path: path}}, Timeout: uint64(timeout / time.Millisecond)}
    config := extism.PluginConfig{EnableWasi: true}
    hosts := []extism.HostFunction{host}
    plugin, err := extism.NewPlugin(context.Background(), manifest, config, hosts)
	if err != nil {
		return err
	}
	// Keep the instance alive: registered handlers capture it.
	_, data, err := plugin.Call("describe", nil)
	if err != nil {
		return fmt.Errorf("describe: %w", err)
	}
	var d pluginapi.Manifest
	if err = json.Unmarshal(data, &d); err != nil {
		return fmt.Errorf("decode plugin manifest: %w", err)
	}
	if err = d.Validate(); err != nil {
        return fmt.Errorf("invalid plugin manifest: %w", err)
    }
    if seenNames[d.Name] { return fmt.Errorf("duplicate plugin name %q", d.Name) }
    seenNames[d.Name] = true
	granted = make(map[string]bool, len(d.Capabilities))
	for _, capability := range d.Capabilities {
		if !allowedPluginCall(capability) {
			return fmt.Errorf("plugin %s requests forbidden capability %q", d.Name, capability)
		}
		granted[capability] = true
	}
	for _, entry := range d.Commands {
		c := core.Command{ID: entry.ID, Path: entry.Path, Description: entry.Description, Risk: string(entry.Risk), Args: entry.Args, InputSchema: entry.InputSchema, OutputSchema: entry.OutputSchema, Examples: func() []string { var out []string; for _, e := range entry.Examples { out = append(out,e.Command) }; return out }()}
		commandID := c.ID
		c.Handler = func(ctx context.Context, input map[string]any) (any, error) {
			payload, _ := json.Marshal(pluginapi.Request{Command: commandID, Input: input})
			callCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout())
			defer cancel()
			callMu.Lock()
			defer callMu.Unlock()
            if err := callCtx.Err(); err != nil { return nil, err }
			exit, resp, err := plugin.CallWithContext(callCtx, "execute", payload)
            if callCtx.Err() != nil || err != nil {
                // A timed-out or trapped wazero module may be closed. Rebuild the
                // instance while holding callMu so the next call is safe.
                _ = plugin.Close(context.Background())
                replacement, replacementErr := extism.NewPlugin(context.Background(), manifest, config, hosts)
                if replacementErr != nil {
                    return nil, fmt.Errorf("plugin call failed (%v); rebuild failed: %w", err, replacementErr)
                }
                plugin = replacement
                if callCtx.Err() != nil { return nil, callCtx.Err() }
                return nil, err
            }
			if err := callCtx.Err(); err != nil { return nil, err }
			if exit != 0 {
				return nil, fmt.Errorf("plugin exit %d", exit)
			}
            if len(resp) > 4<<20 { return nil, fmt.Errorf("plugin response exceeds 4 MiB limit") }
			var value any
			if err = json.Unmarshal(resp, &value); err != nil {
				return nil, err
			}
			return value, nil
		}
		if err = r.Add(c); err != nil {
			return err
		}
	}
	return manager.Add(path, d)
}

// Only explicit, read-only commands may be invoked from WASM.
func allowedPluginCall(id string) bool {
	switch id {
	case "k8s.pods", "k8s.events", "k8s.pod.describe", "k8s.pod.logs", "k8s.nodes", "k8s.namespaces", "k8s.contexts", "k8s.pod.restarts", "k8s.node.describe", "k8s.top.nodes", "k8s.top.pods":
		return true
	}
	return false
}

// A bounded timeout keeps runaway guests from monopolizing the shell.
func pluginCallTimeout() time.Duration {
    if raw := os.Getenv("SUPPORT_PLUGIN_TIMEOUT_MS"); raw != "" {
        if ms, err := strconv.Atoi(raw); err == nil && ms >= 50 && ms <= 120000 {
            return time.Duration(ms) * time.Millisecond
        }
    }
    return 10 * time.Second
}
