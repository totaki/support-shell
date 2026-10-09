package shell

import (
	"strings"
	"testing"
)

func TestConfirmationPrompt(t *testing.T) {
	if got := confirmationPrompt(false); got != "Apply? [y/N] " {
		t.Fatalf("plain confirmation prompt: %q", got)
	}
	got := confirmationPrompt(true)
	if !strings.Contains(got, pink+"Apply?"+reset) || !strings.Contains(got, "[y/N]") {
		t.Fatalf("colored confirmation prompt: %q", got)
	}
}
