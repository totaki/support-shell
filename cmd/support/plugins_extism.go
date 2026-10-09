//go:build extism

package main

import (
	"context"
	"encoding/json"
	"example.com/support-shell/internal/core"
	"fmt"
	extism "github.com/extism/go-sdk"
	"os"
	"time"
)

type descriptor struct {
	Name     string `json:"name"`
	Commands []struct {
		ID          string   `json:"id"`
		Path        string   `json:"path"`
		Description string   `json:"description"`
		Risk        string   `json:"risk"`
		Args        []string `json:"args"`
	} `json:"commands"`
}

func loadPlugins(r *core.Registry) error {
	path := os.Getenv("SUPPORT_PLUGIN")
	if path == "" {
		return nil
	}
	// The extension is intentionally granted only one capability.
	// The host function is restricted to read-only built-in commands.
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
		if !allowedPluginCall(req.Command) {
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
	var d descriptor
	if err = json.Unmarshal(data, &d); err != nil {
		return err
	}
	for _, entry := range d.Commands {
		c := core.Command{ID: entry.ID, Path: entry.Path, Description: entry.Description, Risk: entry.Risk, Args: entry.Args}
		c.Handler = func(ctx context.Context, input map[string]any) (any, error) {
			payload, _ := json.Marshal(map[string]any{"command": c.ID, "input": input})
			callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
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
