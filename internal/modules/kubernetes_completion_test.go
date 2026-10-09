package modules

import (
	"context"
	"example.com/support-shell/internal/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPodCompletionFromKubectl(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
case "$*" in
 *namespaces*) echo '{"items":[{"metadata":{"name":"default"}},{"metadata":{"name":"platform"}}]}' ;;
 *pods*) echo '{"items":[{"metadata":{"name":"controller-0"}},{"metadata":{"name":"coredns-123"}}]}' ;;
 *) exit 1 ;;
esac
`
	path := filepath.Join(dir, "kubectl")
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := core.NewRegistry()
	RegisterKubernetes(r)
	checks := map[string]string{
		"k8s pods pla":                            "k8s pods platform",
		"k8s pod logs pla":                        "k8s pod logs platform",
		"k8s pod logs platform cont":              "k8s pod logs platform controller-0",
		"k8s pod describe platform cont":          "k8s pod describe platform controller-0",
		"k8s pod logs platform controller-0 --ta": "k8s pod logs platform controller-0 --tail=",
	}
	for input, want := range checks {
		suggestions := r.CompleteWithContext(context.Background(), input)
		if !contains(suggestions, want) {
			t.Errorf("completion %q: %v; want %q", input, suggestions, want)
		}
	}
	result, err := r.Execute(context.Background(), "k8s.pods", map[string]any{"namespace": "platform"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(core.JSON(result), "controller-0") {
		t.Fatalf("unexpected result: %v", result)
	}
}
func contains(v []string, target string) bool {
	for _, x := range v {
		if x == target {
			return true
		}
	}
	return false
}
