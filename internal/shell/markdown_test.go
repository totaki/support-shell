package shell

import (
	"strings"
	"testing"
)

func TestMarkdownRendering(t *testing.T) {
	source := "# Diagnosis\n\n## Подтверждено\n- **restartCount**: `126`\n\n| Pod | Count |\n|---|---|\n| calico | 126 |\n\n```bash\nkubectl get pods\n```"
	out := renderMarkdown(source, false)
	for _, want := range []string{"Diagnosis", "Подтверждено", "restartCount", "calico", "126", "kubectl get pods"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	for _, bad := range []string{"**", "```", "|---"} {
		if strings.Contains(out, bad) {
			t.Fatalf("raw markdown %q in %q", bad, out)
		}
	}
	colored := renderMarkdown(source, true)
	if !strings.Contains(colored, "\x1b[") {
		t.Fatal("expected ANSI formatting")
	}
	noEscape := renderMarkdown("hi\x1b[31mbye", false)
	if strings.Contains(noEscape, "\x1b") {
		t.Fatal("terminal escape injection")
	}
}

func TestMarkdownListsAndLiteralHyphens(t *testing.T) {
    cases := []struct{ input, want string }{
        {"- kubelet heartbeat is fresh", "• kubelet heartbeat is fresh"},
        {"  - node Ready", "  • node Ready"},
        {"1. first item", "1. first item"},
        {"-heartbeat", "-heartbeat"},
        {"kubectl logs --previous=true", "kubectl logs --previous=true"},
    }
    for _, tc := range cases {
        t.Run(tc.input, func(t *testing.T) {
            if got := renderMarkdown(tc.input, false); got != tc.want {
                t.Fatalf("renderMarkdown(%q) = %q, want %q", tc.input, got, tc.want)
            }
        })
    }
}
