package shell

import (
 "context"
 "strings"
 "testing"
 "example.com/support-shell/internal/core"
)
func TestCompletionsBuiltinAndRegistry(t *testing.T){
 r:=core.NewRegistry()
 for _,c:=range []core.Command{
  {ID:"k8s.nodes",Path:"k8s nodes",Risk:"read"},
  {ID:"k8s.pods",Path:"k8s pods",Risk:"read"},
  {ID:"plugins.info",Path:"plugins info",Risk:"read",Completer:func(_ context.Context,p string)[]string{
   return []string{"plugins info diagnostics","plugins info network"}
  }},
 } {
  c.Handler=func(context.Context,map[string]any)(any,error){return nil,nil}
  if err:=r.Add(c);err!=nil{t.Fatal(err)}
 }
 s:=Shell{Registry:r}
 cases:=[]struct{prefix,want string}{
  {"he","help"},
  {"help ","help k8s nodes"},
  {"help k8s n","help k8s nodes"},
  {"set f","set format json"},
  {"set format ","set format table"},
  {"set format j","set format json"},
  {"plugins i","plugins info"},
  {"plugins info n","plugins info network"},
  {"k8s n","k8s nodes"},
  {"repo","report"},
  {"agent r","agent reset"},
 }
 for _,tc:=range cases {
  t.Run(tc.prefix,func(t *testing.T){
   options:=s.completions(context.Background(),tc.prefix)
   found:=false
   for _,o:=range options{if o==tc.want{found=true;break}}
   if !found{t.Fatalf("completion for %q missing %q: %v",tc.prefix,tc.want,options)}
   for _,o:=range options{if !strings.HasPrefix(o,tc.prefix){t.Fatalf("invalid candidate %q for %q",o,tc.prefix)}}
  })
 }
}
