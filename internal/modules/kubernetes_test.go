package modules

import (
	"context"
	"example.com/support-shell/internal/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKubectlTextAndErrors(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "kubectl")
	script := `#!/bin/sh
case "$*" in
 "config view -o=json") echo '{"current-context":"dev","contexts":[{"name":"dev","context":{"cluster":"dev-cluster"}}]}' ;;
 *) printf '{"items":[]}\\n'; printf 'Error from server: nodes forbidden\\n' >&2; exit 1;;
esac
`
	if err := os.WriteFile(fake, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := core.NewRegistry()
	RegisterKubernetes(r)
	v, err := r.Execute(context.Background(), "k8s.contexts", nil)
	if err != nil || !strings.Contains(v.(string), "dev-cluster") {
		t.Fatalf("contexts: %v %v", v, err)
	}
	_, err = r.Execute(context.Background(), "k8s.nodes", nil)
	if err == nil || !strings.Contains(err.Error(), "nodes forbidden") || strings.Contains(err.Error(), `"items"`) {
		t.Fatalf("error details not isolated: %v", err)
	}
}

func TestContextCompletionAndApproval(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
case "$*" in
 "config view -o=json") echo '{"current-context":"dev","contexts":[{"name":"dev","context":{"cluster":"dev-c"}},{"name":"prod","context":{"cluster":"prod-c"}}]}' ;;
 "config use-context prod") echo prod > "` + dir + `/switched" ;;
 *) exit 10 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "kubectl"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := core.NewRegistry()
	RegisterKubernetes(r)
	matches := r.CompleteWithContext(context.Background(), "k8s context use pr")
	if len(matches) != 1 || matches[0] != "k8s context use prod" {
		t.Fatalf("completion: %v", matches)
	}
	_, err := r.Execute(context.Background(), "k8s.context.use", map[string]any{"name": "prod"})
	if err == nil {
		t.Fatal("unapproved switch was allowed")
	}
	if _, err = os.Stat(filepath.Join(dir, "switched")); !os.IsNotExist(err) {
		t.Fatal("side effect before approval")
	}
	result, err := r.ExecuteApproved(context.Background(), "k8s.context.use", map[string]any{"name": "prod"})
	if err != nil || !strings.Contains(result.(string), "prod") {
		t.Fatalf("approved switch: %v %v", result, err)
	}
}
