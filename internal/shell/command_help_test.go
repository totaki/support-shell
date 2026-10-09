package shell

import (
 "strings"
 "testing"
 "example.com/support-shell/internal/core"
)
func TestCommandHelpUsesMetadata(t *testing.T) {
 cmd:=core.Command{ID:"network.health",Path:"network health",Description:"Node diagnostic findings",Risk:"read",Examples:[]string{"network health"},InputSchema:map[string]any{"type":"object","properties":map[string]any{}}}
 got:=commandHelp(cmd)
 for _,part:=range []string{"network health","Node diagnostic findings","Examples:","Input schema:"} {
  if !strings.Contains(got,part) { t.Fatalf("missing %q in %s",part,got) }
 }
}
