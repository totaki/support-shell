package shell

import (
	"strings"
	"testing"
)

func TestKubernetesTableRendering(t *testing.T) {
	cs := map[string]any{"ready": false, "restartCount": float64(8), "state": map[string]any{"waiting": map[string]any{"reason": "CrashLoopBackOff"}}}
	pod := map[string]any{"metadata": map[string]any{"name": "controller-0"}, "status": map[string]any{"phase": "Running", "containerStatuses": []any{cs}}}
	out := renderCommand("k8s.pods", map[string]any{"items": []any{pod}})
	for _, want := range []string{"NAME", "READY", "RESTARTS", "controller-0", "CrashLoopBackOff", "8"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}
