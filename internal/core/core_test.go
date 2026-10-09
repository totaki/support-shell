package core

import (
	"context"
	"testing"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	r.Add(Command{ID: "test", Path: "cluster health", Risk: "read", Args: []string{"name"}, Handler: func(_ context.Context, a map[string]any) (any, error) { return a, nil }})
	c, args, ok := r.Resolve("cluster health prod")
	if !ok || c.ID != "test" || len(args) != 1 {
		t.Fatal("resolve failed")
	}
	in, e := ParseInput(c, args)
	if e != nil || in["name"] != "prod" {
		t.Fatalf("parse: %v %v", in, e)
	}
	if len(r.Complete("clu")) != 1 {
		t.Fatal("completion")
	}
}
func TestMutationDenied(t *testing.T) {
	r := NewRegistry()
	r.Add(Command{ID: "x", Path: "x", Risk: "mutate", Handler: func(context.Context, map[string]any) (any, error) { t.Fatal("mutation executed"); return nil, nil }})
	if _, e := r.Execute(context.Background(), "x", nil); e == nil {
		t.Fatal("expected denial")
	}
}
