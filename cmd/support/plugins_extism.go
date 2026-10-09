//go:build extism

package main

import (
	"context"
	"encoding/json"
	"example.com/support-shell/internal/core"
	"example.com/support-shell/pkg/pluginapi"
	"fmt"
	extism "github.com/extism/go-sdk"
	"os"
	"strings"
	"path/filepath"
	"sync"
	"time"
)


func loadPlugins(r *core.Registry) error {
    paths := strings.TrimSpace(os.Getenv("SUPPORT_PLUGINS"))
    if paths == "" {
        paths = strings.TrimSpace(os.Getenv("SUPPORT_PLUGIN"))
    }
    if paths == "" { return nil }
    seenPaths := make(map[string]bool)
    seenNames := make(map[string]bool)
    for _, item := range strings.Split(paths, ",") {
        path := strings.TrimSpace(item)
        if path == "" { return fmt.Errorf("empty WASM plugin path in SUPPORT_PLUGINS") }
        abs, err := filepath.Abs(path)
        if err != nil { return err }
        if seenPaths[abs] { return fmt.Errorf("duplicate WASM plugin path: %s", path) }
        seenPaths[abs] = true
        if err := loadSinglePlugin(r, abs, seenNames); err != nil {
            return fmt.Errorf("load WASM plugin %s: %w", path, err)
        }
    }
    return nil
}

func loadSinglePlugin(r *core.Registry, path string, seenNames map[string]bool) error {
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
	plugin, err := extism.NewPlugin(context.Background(), extism.Manifest{Wasm: []extism.Wasm{extism.WasmFile{Path: path}}}, extism.PluginConfig{EnableWasi: true}, []extism.HostFunction{host})
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
		c := core.Command{ID: entry.ID, Path: entry.Path, Description: entry.Description, Risk: string(entry.Risk), Args: entry.Args}
		commandID := c.ID
		c.Handler = func(ctx context.Context, input map[string]any) (any, error) {
			payload, _ := json.Marshal(pluginapi.Request{Command: commandID, Input: input})
			callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			callMu.Lock()
			defer callMu.Unlock()
			exit, resp, err := plugin.CallWithContext(callCtx, "execute", payload)
			if err != nil {
				return nil, err
			}
			if exit != 0 {
				return nil, fmt.Errorf("plugin exit %d", exit)
			}
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
	return nil
}

// Only explicit, read-only commands may be invoked from WASM.
func allowedPluginCall(id string) bool {
	switch id {
	case "k8s.pods", "k8s.events", "k8s.pod.describe", "k8s.pod.logs", "k8s.nodes", "k8s.namespaces", "k8s.contexts", "k8s.pod.restarts", "k8s.node.describe", "k8s.top.nodes", "k8s.top.pods":
		return true
	}
	return false
}
