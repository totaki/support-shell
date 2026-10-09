package modules

import (
	"context"
	"example.com/support-shell/internal/core"
	"fmt"
)

func RegisterDemo(r *core.Registry) {
	for _, c := range []core.Command{
		{ID: "demo.clusters.list", Path: "clusters", Description: "List demo clusters", Risk: "read", Handler: func(ctx context.Context, in map[string]any) (any, error) {
			return map[string]any{"items": []any{map[string]any{"name": "prod-01", "status": "Ready"}, map[string]any{"name": "stage-01", "status": "Ready"}}}, nil
		}},
		{ID: "demo.cluster.health", Path: "cluster health", Description: "Inspect demo cluster", Risk: "read", Args: []string{"cluster"}, Handler: func(ctx context.Context, in map[string]any) (any, error) {
			name, _ := in["cluster"].(string)
			if name == "" {
				return nil, fmt.Errorf("cluster argument required")
			}
			return map[string]any{"cluster": name, "status": "Degraded", "warning": "controller-0 is restarting"}, nil
		}},
		{ID: "demo.cluster.restart", Path: "cluster restart", Description: "Mutation blocked until approval implemented", Risk: "mutate", Args: []string{"cluster"}, Handler: func(ctx context.Context, in map[string]any) (any, error) { return "RESTARTED", nil }},
	} {
		if err := r.Add(c); err != nil {
			panic(err)
		}
	}
}
