package agent

import (
 "context"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "example.com/support-shell/internal/core"
)

func TestAgentFailedModelTurnDoesNotPolluteHistory(t *testing.T) {
 cases:=[]struct{name string;status int;body string}{
  {"server error",503,"unavailable"},
  {"malformed json",200,"not-json"},
  {"empty choices",200,`{"choices":[]}`},
 }
 for _,tc:=range cases {t.Run(tc.name,func(t *testing.T){
  server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,_ *http.Request){w.WriteHeader(tc.status);_,_=w.Write([]byte(tc.body))}))
  defer server.Close()
  a:=New(core.NewRegistry());a.URL=server.URL;a.Key="test"
  a.Messages=[]message{{Role:"system",Content:"existing"}}
  if _,err:=a.Ask(context.Background(),"investigate");err==nil{t.Fatal("expected error")}
  if len(a.Messages)!=1||a.Messages[0].Content!="existing"{t.Fatalf("failed turn polluted history: %+v",a.Messages)}
 })}
}

func TestAgentRejectsMalformedToolArguments(t *testing.T) {
 requests,executions:=0,0
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,_ *http.Request){
  requests++
  if requests==1{
   _,_=w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"bad","type":"function","function":{"name":"safe_read","arguments":"NOT JSON"}}]}}]}`))
   return
  }
  _,_=w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"done"}}]}`))
 }))
 defer server.Close()
 registry:=core.NewRegistry()
 _=registry.Add(core.Command{ID:"safe.read",Path:"safe read",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){executions++;return "ok",nil}})
 a:=New(registry);a.URL=server.URL;a.Key="test"
 answer,err:=a.Ask(context.Background(),"check")
 if err!=nil||answer!="done"{t.Fatalf("answer=%q err=%v",answer,err)}
 if executions!=0{t.Fatal("executed tool with malformed JSON")}
 found:=false
 for _,m:=range a.Messages {if m.Role=="tool"{v,_:=m.Content.(string);found=strings.Contains(v,"JSON object")}}
 if !found{t.Fatalf("missing argument error in tool response: %+v",a.Messages)}
}
