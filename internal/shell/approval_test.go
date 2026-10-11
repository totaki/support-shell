package shell

import (
 "context"
 "testing"

 "example.com/support-shell/internal/core"
 "example.com/support-shell/internal/provider"
)

type connectionTestAgent struct{ resets int }
func(a *connectionTestAgent) Reset(){a.resets++}
func(a *connectionTestAgent) Ask(context.Context,string)(string,error){return "",nil}

func TestConnectionSelectionConfirmation(t *testing.T) {
 providers:=provider.NewRegistry()
 catalog:=provider.NewCatalog("kubernetes")
 catalog.SetConnectionDiscovery(func(context.Context)([]provider.Connection,error){
  return []provider.Connection{
   {Provider:"kubernetes",ID:"dev",Target:"dev",Default:true},
   {Provider:"kubernetes",ID:"prod",Target:"prod"},
  },nil
 })
 _=catalog.Add(provider.Operation{ID:"provider.kubernetes.dummy",Path:"k8s dummy",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){return nil,nil}})
 if err:=providers.Register(catalog);err!=nil{t.Fatal(err)}
 for _,tc:=range []struct{answer,want string}{
  {"n","dev"},{"","dev"},{"yes","dev"},{"Y","prod"},{"y","prod"},
 } {
  t.Run(tc.answer,func(t *testing.T){
   session:=provider.NewSession()
   if err:=session.SelectDefaults(context.Background(),providers);err!=nil{t.Fatal(err)}
   agent:=&connectionTestAgent{}
   sh:=&Shell{Registry:core.NewRegistry(),Providers:providers,Connections:session,Agent:agent,PendingContext:"kubernetes/prod"}
   sh.applyContextDecision(context.Background(),tc.answer)
   if got:=session.Current("kubernetes");got!=tc.want{t.Fatalf("answer=%q current=%q want=%q",tc.answer,got,tc.want)}
   if sh.PendingContext!=""{t.Fatalf("pending not cleared: %q",sh.PendingContext)}
   wantResets:=0
   if tc.want=="prod"{wantResets=1}
   if agent.resets!=wantResets{t.Fatalf("reset count=%d want=%d",agent.resets,wantResets)}
  })
 }
}
func TestCurrentContextName(t *testing.T){
 output:="CURRENT   NAME   CLUSTER\n*         dev    dev-cluster\n          prod   prod-cluster"
 if got:=currentContextName(output);got!="dev"{t.Fatal(got)}
}
