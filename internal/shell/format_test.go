package shell

import (
	"strings"
	"testing"
)

func TestFormatResult(t *testing.T) {
	got := formatResult("COL1 COL2\nA B")
	if got != "COL1 COL2\nA B" {
		t.Fatalf("string result was JSON-quoted: %q", got)
	}
	data := formatResult(map[string]any{"status": "ok"})
	if !strings.Contains(data, "\n") || !strings.Contains(data, `"status": "ok"`) {
		t.Fatalf("JSON result not pretty-printed: %q", data)
	}
}
