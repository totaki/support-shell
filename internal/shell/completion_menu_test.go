package shell

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestCompletionSelection(t *testing.T) {
	choices := []string{"k8s pods default", "k8s pods platform", "k8s pods kube-system"}
	for _, tc := range []struct {
		name, keys, want string
		ok               bool
	}{
		{"tab-next", "\t\r", choices[1], true},
		{"down", "\x1b[B\r", choices[1], true},
		{"up-wrap", "\x1b[A\r", choices[2], true},
		{"escape", "\x1b", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			got, ok := selectCompletion(bufio.NewReader(strings.NewReader(tc.keys)), &out, choices, false)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("got %q,%v expected %q,%v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestCompletionMatches(t *testing.T) {
	got := completionMatches("k8s p", []string{"k8s pods", "k8s pod logs", "cluster"})
	if len(got) != 2 {
		t.Fatalf("matches: %v", got)
	}
}
