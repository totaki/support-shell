package shell

import (
	"os"
	"strings"
	"testing"
)

func TestCompactToolTrace(t *testing.T) {
	got := compactInput(map[string]any{"namespace": "kube-system", "pod": "controller-0"})
	if got != "[namespace=kube-system pod=controller-0]" {
		t.Fatal(got)
	}
}
func TestSaveReport(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := saveReport("## Подтверждено\nЕсть рестарты", "test/context")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), "Есть рестарты") {
		t.Fatalf("read: %v", err)
	}
	if perm := filePerm(path); perm != 0600 {
		t.Fatalf("permissions: %o", perm)
	}
}
func filePerm(path string) os.FileMode {
	s, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return s.Mode().Perm()
}
