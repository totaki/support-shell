package kubernetes

import (
 "testing"
 "example.com/support-shell/internal/core"
 "example.com/support-shell/internal/provider"
)

func registerForTest(t *testing.T, commands *core.Registry){
 t.Helper()
 providers:=provider.NewRegistry()
 if err:=providers.Register(New());err!=nil{t.Fatal(err)}
 if err:=providers.BindTools(commands);err!=nil{t.Fatal(err)}
}

func TestProviderCatalog(t *testing.T){
 providers:=provider.NewRegistry()
 if err:=providers.Register(New());err!=nil{t.Fatal(err)}
 if len(providers.IDs())!=1||providers.IDs()[0]!="kubernetes"{t.Fatalf("providers=%v",providers.IDs())}
 if !providers.Readable("provider.kubernetes.pods"){t.Fatal("pods capability missing")}
 if providers.Readable("provider.kubernetes.context.use"){t.Fatal("context change exposed as read-only")}
}
