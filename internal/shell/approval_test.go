package shell

import (
	"context"
	"example.com/support-shell/internal/core"
	"testing"
)

func TestContextApproval(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int
	}{{"", 0}, {"n", 0}, {"yes", 0}, {"Y", 1}, {"y", 1}} {
		called := 0
		reg := core.NewRegistry()
		if err := reg.Add(core.Command{ID: "k8s.context.use", Path: "k8s context use", Risk: "mutate", Handler: func(_ context.Context, _ map[string]any) (any, error) { called++; return "ok", nil }}); err != nil {
			t.Fatal(err)
		}
		sh := &Shell{Registry: reg, PendingContext: "prod"}
		sh.applyContextDecision(context.Background(), tc.input)
		if called != tc.want || sh.PendingContext != "" {
			t.Fatalf("input=%q called=%d pending=%q", tc.input, called, sh.PendingContext)
		}
	}
}
func TestCurrentContextName(t *testing.T) {
	output := "CURRENT   NAME   CLUSTER\n*         dev    dev-cluster\n          prod   prod-cluster"
	if got := currentContextName(output); got != "dev" {
		t.Fatal(got)
	}
}
