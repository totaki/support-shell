package provider

import (
 "context"
 "testing"
)

func TestConnectionSessionsAreIsolatedAndBoundToContext(t *testing.T){
 catalog:=NewCatalog("kubernetes")
 catalog.SetConnectionDiscovery(func(context.Context)([]Connection,error){
  return []Connection{
   {Provider:"kubernetes",ID:"dev",Target:"dev",Default:true},
   {Provider:"kubernetes",ID:"prod",Target:"prod"},
  },nil
 })
 if err:=catalog.Add(Operation{ID:"provider.kubernetes.nodes",Path:"k8s nodes",Risk:"read",Handler:func(ctx context.Context,_ map[string]any)(any,error){
  name,_:=ConnectionFromContext(ctx,"kubernetes")
  return name,nil
 }});err!=nil{t.Fatal(err)}
 registry:=NewRegistry()
 if err:=registry.Register(catalog);err!=nil{t.Fatal(err)}
 alice,bob:=NewSession(),NewSession()
 for _,s:=range []*Session{alice,bob}{
  if err:=s.SelectDefaults(context.Background(),registry);err!=nil{t.Fatal(err)}
 }
 snapshot:=alice.Bind(context.Background())
 if err:=alice.Select(context.Background(),registry,"kubernetes","prod");err!=nil{t.Fatal(err)}
 if got:=bob.Current("kubernetes");got!="dev"{t.Fatalf("other session changed: %q",got)}
 if got,_:=ConnectionFromContext(snapshot,"kubernetes");got!="dev"{t.Fatalf("previous context mutated: %q",got)}
 if got,_:=ConnectionFromContext(alice.Bind(context.Background()),"kubernetes");got!="prod"{t.Fatalf("selection not propagated: %q",got)}
 if err:=alice.Select(context.Background(),registry,"kubernetes","not-found");err==nil{t.Fatal("unknown connection allowed")}
 if got:=alice.Current("kubernetes");got!="prod"{t.Fatalf("selection changed after error: %q",got)}
 result,err:=registry.ExecuteRead(alice.Bind(context.Background()),"provider.kubernetes.nodes",nil)
 if err!=nil||result!="prod"{t.Fatalf("provider execution=%v error=%v",result,err)}
}
func TestConnectionContextSnapshotDoesNotLeakAcrossProviders(t *testing.T){
 a:=WithConnection(context.Background(),"kubernetes","dev")
 b:=WithConnection(a,"postgres","local")
 if _,ok:=ConnectionFromContext(a,"postgres");ok{t.Fatal("original context unexpectedly modified")}
 if name,ok:=ConnectionFromContext(b,"kubernetes");!ok||name!="dev"{t.Fatalf("lost Kubernetes selection: %q",name)}
 if name,ok:=ConnectionFromContext(b,"postgres");!ok||name!="local"{t.Fatalf("lost PostgreSQL selection: %q",name)}
}

func TestConnectionIDsWithSlashes(t *testing.T){
 catalog:=NewCatalog("kubernetes")
 target:="arn:aws:eks:eu-west-1:123456789:cluster/production"
 catalog.SetConnectionDiscovery(func(context.Context)([]Connection,error){
  return []Connection{{Provider:"kubernetes",ID:target,Target:target,Default:true}},nil
 })
 _=catalog.Add(Operation{ID:"provider.kubernetes.test",Path:"k8s test",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){return nil,nil}})
 registry:=NewRegistry()
 if err:=registry.Register(catalog);err!=nil{t.Fatal(err)}
 session:=NewSession()
 if err:=session.Select(context.Background(),registry,"kubernetes",target);err!=nil{t.Fatal(err)}
 if got:=session.Current("kubernetes");got!=target{t.Fatalf("got %q",got)}
}
