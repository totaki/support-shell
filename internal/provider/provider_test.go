package provider

import (
 "context"
 "strings"
 "testing"

 "example.com/support-shell/internal/core"
)

func TestProviderRegistryAndCapabilities(t *testing.T){
 r:=NewRegistry()
 c:=NewCatalog("sample")
 calls:=0
 if err:=c.Add(Operation{ID:"provider.sample.read",Path:"sample read",Risk:"read",InputSchema:map[string]any{"type":"object","properties":map[string]any{"name":map[string]any{"type":"string"}},"required":[]any{"name"},"additionalProperties":false},Handler:func(_ context.Context,in map[string]any)(any,error){calls++;return in["name"],nil}});err!=nil{t.Fatal(err)}
 if err:=c.Add(Operation{ID:"provider.sample.write",Path:"sample write",Risk:"write",Handler:func(context.Context,map[string]any)(any,error){calls++;return nil,nil}});err!=nil{t.Fatal(err)}
 if err:=r.Register(c);err!=nil{t.Fatal(err)}
 if err:=r.Register(c);err==nil{t.Fatal("duplicate provider accepted")}
 if !r.Readable("provider.sample.read")||r.Readable("provider.sample.write"){t.Fatal("read boundary incorrect")}
 if _,err:=r.ExecuteRead(context.Background(),"provider.sample.write",nil);err==nil{t.Fatal("write capability accessible to plugin")}
 if _,err:=r.ExecuteRead(context.Background(),"provider.sample.read",map[string]any{"name":2});err==nil{t.Fatal("invalid input accepted")}
 value,err:=r.ExecuteRead(context.Background(),"provider.sample.read",map[string]any{"name":"ok"})
 if err!=nil||value!="ok"{t.Fatalf("value=%v err=%v",value,err)}
 if calls!=1{t.Fatalf("unexpected executions=%d",calls)}
 registry:=core.NewRegistry()
 if err:=r.BindTools(registry);err!=nil{t.Fatal(err)}
 if _,err:=registry.Execute(context.Background(),"provider.sample.write",nil);err==nil{t.Fatal("write bypassed core")}
}
func TestProviderRejectsForeignOperations(t *testing.T){
 for _,id:=range []string{"outside","provider.other.read"}{
  c:=NewCatalog("sample")
  _=c.Add(Operation{ID:id,Path:"test",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){return nil,nil}})
  if err:=NewRegistry().Register(c);err==nil{t.Fatal("accepted "+id)}
 }
 c:=NewCatalog("invalid")
 if err:=c.Add(Operation{ID:"provider.invalid.read",Path:"read",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){return nil,nil}});err!=nil{t.Fatal(err)}
 if err:=c.Add(Operation{ID:"provider.invalid.read",Path:"other",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){return nil,nil}});err==nil{t.Fatal("duplicate operation accepted")}
 if _,err:=NewRegistry().ExecuteRead(context.Background(),"missing",nil);err==nil||!strings.Contains(err.Error(),"unknown"){t.Fatalf("unknown error=%v",err)}
}
