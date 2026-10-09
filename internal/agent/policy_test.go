package agent

import (
	"context"
	"encoding/json"
	"example.com/support-shell/internal/core"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestToolBudgetAndReport(t *testing.T) {
	n, executed := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		var payload struct {
			Tools    []any     `json:"tools"`
			Messages []message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode: %v", err)
		}
		if n == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"safe","arguments":"{}"}},{"id":"2","type":"function","function":{"name":"safe","arguments":"{}"}}]}}]}`))
			return
		}
		if len(payload.Tools) != 0 {
			t.Errorf("tools should be withheld after budget exhausted")
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Подтверждено: checked"}}]}`))
	}))
	defer server.Close()
	r := core.NewRegistry()
	_ = r.Add(core.Command{ID: "safe", Path: "safe", Risk: "read", Handler: func(context.Context, map[string]any) (any, error) { executed++; return "ok", nil }})
	a := New(r)
	a.URL = server.URL
	a.Key = "test"
	a.MaxCalls = 1
	a.SetContext("default/rnd")
	out, err := a.Ask(context.Background(), "проверь")
	if err != nil || !strings.Contains(out, "checked") || executed != 1 || n != 2 {
		t.Fatalf("out=%q err=%v executed=%d reqs=%d", out, err, executed, n)
	}
	if a.Report() != out || a.ContextLabel() != "default/rnd" {
		t.Fatalf("report or context missing")
	}
	a.Reset()
	if a.Report() != "" {
		t.Fatal("reset must clear report")
	}
}
