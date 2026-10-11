package shell

import (
	"strings"
	"testing"
)

func TestPromptContext(t *testing.T) {
	s := &Shell{CurrentContext: "prod-cluster"}
	if got := s.prompt(false); got != "support ❯ " {
		t.Fatalf("prompt=%q", got)
	}
	if got := s.prompt(true); !strings.Contains(got, "support") || strings.Contains(got, "prod-cluster") {
		t.Fatalf("colored prompt=%q", got)
	}
}
