package modules

import (
	"context"
	"encoding/json"
	"example.com/support-shell/internal/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPodRestartsStructured(t *testing.T) {
	d := t.TempDir()
	script := filepath.Join(d, "kubectl")
	pod := `{"spec":{"nodeName":"master-1","containers":[{"name":"calico","livenessProbe":{"timeoutSeconds":10}}]},"status":{"phase":"Running","containerStatuses":[{"name":"calico","ready":true,"restartCount":126,"lastState":{"terminated":{"exitCode":1,"reason":"Error","finishedAt":"2026-09-28T10:00:00Z"}}}]}}`
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' '"+pod+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", d+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := core.NewRegistry()
	RegisterKubernetes(r)
	out, err := r.Execute(context.Background(), "k8s.pod.restarts", map[string]any{"namespace": "kube-system", "pod": "calico"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(out)
	if !strings.Contains(string(b), `"restartCount":126`) || !strings.Contains(string(b), `"livenessProbe"`) || !strings.Contains(string(b), `"lastTermination"`) {
		t.Fatalf("unexpected %s", b)
	}
}
func TestLogsPreviousFlags(t *testing.T) {
	d := t.TempDir()
	script := filepath.Join(d, "kubectl")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", d+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := core.NewRegistry()
	RegisterKubernetes(r)
	out, err := r.Execute(context.Background(), "k8s.pod.logs", map[string]any{"namespace": "kube-system", "pod": "calico", "previous": "true", "timestamps": "true"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(out)
	if !strings.Contains(string(b), "--previous=true") || !strings.Contains(string(b), "--timestamps=true") {
		t.Fatalf("flags absent: %s", b)
	}
}
