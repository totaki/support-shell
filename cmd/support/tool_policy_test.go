package main

import (
 "os"
 "path/filepath"
 "testing"
 "example.com/support-shell/internal/core"
)

func TestConfigureToolPolicyYAMLAndEnv(t *testing.T) {
 dir:=t.TempDir()
 path:=filepath.Join(dir,"policy.yaml")
 data:=[]byte("deny:\n  cli:\n    - tool.local\n  agent:\n    - tool.secret\n  mcp:\n    - tool.logs\n")
 if err:=os.WriteFile(path,data,0600);err!=nil{t.Fatal(err)}
 t.Setenv("SUPPORT_POLICY_FILE",path)
 t.Setenv("SUPPORT_DENY_MCP","tool.extra,tool.logs")
 r:=core.NewRegistry()
 if err:=configureToolPolicy(r);err!=nil{t.Fatal(err)}
 for _,tc:=range []struct{surface core.Surface;id string;allow bool}{
  {core.SurfaceCLI,"tool.local",false},
  {core.SurfaceAgent,"tool.secret",false},
  {core.SurfaceMCP,"tool.logs",false},
  {core.SurfaceMCP,"tool.extra",false},
  {core.SurfaceCLI,"tool.extra",true},
 } {
  c:=core.Command{ID:tc.id,Risk:"read"}
  if got:=r.Allowed(tc.surface,c);got!=tc.allow{t.Errorf("%s/%s allowed=%v want %v",tc.surface,tc.id,got,tc.allow)}
 }
}
func TestConfigureToolPolicyRejectsBadYAML(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"policy.yaml")
 _=os.WriteFile(path,[]byte("deny:\n  unknown:\n    - x\n"),0600)
 t.Setenv("SUPPORT_POLICY_FILE",path)
 if err:=configureToolPolicy(core.NewRegistry());err==nil{t.Fatal("unknown surface accepted")}
}
