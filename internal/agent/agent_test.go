package agent

import (
	"context"
	"encoding/json"
	"example.com/support-shell/internal/core"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestToolCalling(t *testing.T) {
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		var req struct {
			Messages []struct {
				Role string `json:"role"`
			} `json:"messages"`
			Tools []struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if count == 1 {
			if len(req.Tools) != 1 {
				t.Errorf("tool count %d", len(req.Tools))
			} else if req.Tools[0].Function.Name != "demo_safe" {
				t.Errorf("invalid tool name %q", req.Tools[0].Function.Name)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call1","type":"function","function":{"name":"demo_safe","arguments":"{}"}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"checked"}}]}`))
	}))
	defer srv.Close()
	registry := core.NewRegistry()
	_ = registry.Add(core.Command{ID: "demo.safe", Path: "safe", Risk: "read", Handler: func(context.Context, map[string]any) (any, error) { return "ok", nil }})
	_ = registry.Add(core.Command{ID: "demo.danger", Path: "danger", Risk: "mutate", Handler: func(context.Context, map[string]any) (any, error) { t.Fatal("mutation executed"); return nil, nil }})
	a := New(registry)
	a.URL = srv.URL
	a.Key = "test"
	text, err := a.Ask(context.Background(), "check")
	if err != nil || text != "checked" || count != 2 {
		t.Fatalf("got %q %v count %d", text, err, count)
	}
}

func TestSessionHeaderStableAndReset(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("x-opencode-session"))
		if r.Header.Get("User-Agent") != "support-shell/0.5" {
			t.Errorf("missing user agent")
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()
	a := New(core.NewRegistry())
	a.URL = srv.URL
	a.Key = "test"
	for range 2 {
		if _, err := a.Ask(context.Background(), "hello"); err != nil {
			t.Fatal(err)
		}
	}
	if len(seen) != 2 || seen[0] == "" || seen[0] != seen[1] {
		t.Fatalf("session not stable: %v", seen)
	}
	a.Reset()
	if _, err := a.Ask(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if seen[2] == seen[1] {
		t.Fatalf("session ID not rotated on reset: %v", seen)
	}
}

func TestToolAlias(t *testing.T) {
	for input, want := range map[string]string{"demo.cluster.health": "demo_cluster_health", "kubernetes.pods.list": "kubernetes_pods_list", "abc-foo": "abc-foo"} {
		if got := toolAlias(input); got != want {
			t.Errorf("%q => %q, want %q", input, got, want)
		}
	}
}

func TestToolTrace(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"k8s_nodes","arguments":"{}"}}]}}]}`))
		} else {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"checked"}}]}`))
		}
	}))
	defer srv.Close()
	reg := core.NewRegistry()
	_ = reg.Add(core.Command{ID: "k8s.nodes", Path: "k8s nodes", Risk: "read", Handler: func(context.Context, map[string]any) (any, error) { return map[string]any{"items": []any{}}, nil }})
	a := New(reg)
	a.URL = srv.URL
	a.Key = "test"
	var trace []ToolEvent
	a.OnTool = func(e ToolEvent) { trace = append(trace, e) }
	if _, err := a.Ask(context.Background(), "check"); err != nil {
		t.Fatal(err)
	}
	if len(trace) != 1 || trace[0].Command != "k8s.nodes" || trace[0].Err != nil {
		t.Fatalf("unexpected trace: %+v", trace)
	}
}
