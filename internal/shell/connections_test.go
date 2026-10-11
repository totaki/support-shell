package shell

import (
 "context"
 "testing"

 "example.com/support-shell/internal/core"
 "example.com/support-shell/internal/provider"
)

func TestInvalidConfiguredConnectionDoesNotFallBack(t *testing.T) {
 t.Setenv("SUPPORT_KUBERNETES_CONNECTION","missing")
 catalog:=provider.NewCatalog("kubernetes")
 catalog.SetConnectionDiscovery(func(context.Context)([]provider.Connection,error){
  return []provider.Connection{{Provider:"kubernetes",ID:"dev",Target:"dev",Default:true}},nil
 })
 if err:=catalog.Add(provider.Operation{ID:"provider.kubernetes.test",Path:"k8s test",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){return nil,nil}});err!=nil{t.Fatal(err)}
 providers:=provider.NewRegistry()
 if err:=providers.Register(catalog);err!=nil{t.Fatal(err)}
 shell:=&Shell{Registry:core.NewRegistry(),Providers:providers,Connections:provider.NewSession()}
 scoped:=shell.connectionContext(context.Background())
 if value,ok:=provider.ConnectionFromContext(scoped,"kubernetes");ok{t.Fatalf("unexpected fallback connection %q",value)}
 if shell.CurrentContext!=""{t.Fatalf("unexpected prompt scope %q",shell.CurrentContext)}
}
