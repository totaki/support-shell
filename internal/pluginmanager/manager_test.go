package pluginmanager

import (
 "context"
 "testing"
 "example.com/support-shell/internal/core"
 "example.com/support-shell/pkg/pluginapi"
)
func TestRegisterAndLookup(t *testing.T){
 m:=New();r:=core.NewRegistry()
 if err:=m.Register(r);err!=nil{t.Fatal(err)}
 mft:=pluginapi.Manifest{APIVersion:pluginapi.APIVersion,Name:"diagnostics",Version:"0.1.0",Capabilities:[]string{"k8s.pods"},Commands:[]pluginapi.Command{{ID:"diagnostics.inspect",Path:"diagnose k8s",Risk:pluginapi.RiskRead}}}
 if err:=m.Add("/tmp/diagnostics.wasm",mft);err!=nil{t.Fatal(err)}
 if err:=m.Add("/tmp/other.wasm",mft);err==nil{t.Fatal("expected duplicate plugin error")}
 out,err:=r.Execute(context.Background(),"plugins.list",nil);if err!=nil{t.Fatal(err)}
 plugins:=out.(map[string]any)["plugins"].([]Plugin)
 if len(plugins)!=1||plugins[0].Name!="diagnostics"{t.Fatalf("unexpected plugins: %+v",plugins)}
 detail,err:=r.Execute(context.Background(),"plugins.info",map[string]any{"name":"diagnostics"})
 if err!=nil||detail.(Plugin).Path!="/tmp/diagnostics.wasm"{t.Fatalf("unexpected detail: %v / %v",detail,err)}
 if _,err:=r.Execute(context.Background(),"plugins.info",map[string]any{"name":"missing"});err==nil{t.Fatal("expected missing plugin error")}
}
