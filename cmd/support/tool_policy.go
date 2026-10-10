package main

import (
 "os"
 "strings"
 "example.com/support-shell/internal/core"
)

func configureToolPolicy(r *core.Registry) {
 deny:=map[core.Surface]map[string]bool{}
 for surface,key:=range map[core.Surface]string{core.SurfaceCLI:"SUPPORT_DENY_CLI",core.SurfaceAgent:"SUPPORT_DENY_AGENT",core.SurfaceMCP:"SUPPORT_DENY_MCP"} {
  deny[surface]=map[string]bool{}
  for _,v:=range strings.Split(os.Getenv(key),",") {if id:=strings.TrimSpace(v);id!=""{deny[surface][id]=true}}
 }
 r.SetToolPolicy(core.ToolPolicy{Deny:deny})
}
