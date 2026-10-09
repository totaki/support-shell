package agent

import (
	"strings"
	"testing"
	"time"
)

func TestCompactToolResult(t *testing.T) {
	small := compactToolResult(map[string]string{"status": "ok"})
	if !strings.Contains(small, "ok") || strings.Contains(small, "truncated") {
		t.Fatalf("small result: %s", small)
	}
	large := compactToolResult(map[string]string{"logs": strings.Repeat("x", 30000)})
	if len(large) > 7000 || !strings.Contains(large, `"truncated": true`) {
		t.Fatalf("large result not truncated, len=%d", len(large))
	}
}

func TestModelTimeoutConfig(t *testing.T) {
	t.Setenv("SUPPORT_AGENT_TIMEOUT_SECONDS", "")
	if modelTimeout() != 180*time.Second {
		t.Fatal("default timeout")
	}
	t.Setenv("SUPPORT_AGENT_TIMEOUT_SECONDS", "240")
	if modelTimeout() != 240*time.Second {
		t.Fatal("configured timeout")
	}
}
