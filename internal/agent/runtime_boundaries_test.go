package agent

import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "example.com/support-shell/internal/core"
)

func TestAgentToolBudget(t *testing.T) {
 requests, executions:=0,0
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  requests++
  var body struct{Tools []any `json:"tools"`}
  if err:=json.NewDecoder(r.Body).Decode(&body);err!=nil{t.Error(err)}
  if requests==1 {
   if len(body.Tools)!=1 {t.Errorf("first request tools=%d",len(body.Tools))}
   _,_=w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"a","type":"function","function":{"name":"safe_read","arguments":"{}"}},{"id":"b","type":"function","function":{"name":"safe_read","arguments":"{}"}}]}}]}`))
   return
  }
  if len(body.Tools)!=0 {t.Errorf("tools offered after budget used: %d",len(body.Tools))}
  _,_=w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"finished"}}]}`))
 }))
 defer srv.Close()
 reg:=core.NewRegistry()
 if err:=reg.Add(core.Command{ID:"safe.read",Path:"safe read",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){executions++;return "ok",nil}});err!=nil{t.Fatal(err)}
 a:=New(reg);a.URL=srv.URL;a.Key="test";a.MaxCalls=1
 answer,err:=a.Ask(context.Background(),"check")
 if err!=nil||answer!="finished" {t.Fatalf("answer=%q err=%v",answer,err)}
 if requests!=2||executions!=1 {t.Fatalf("requests=%d tool executions=%d",requests,executions)}
 if len(a.Messages)!=6 {t.Fatalf("message count=%d; expected system/user/assistant/two results/final",len(a.Messages))}
 if result,ok:=a.Messages[4].Content.(string);!ok||!strings.Contains(result,"tool budget exceeded"){t.Fatalf("budget result=%v",a.Messages[4].Content)}
}

func TestAgentCancellationRollsBackHistory(t *testing.T) {
 ctx,cancel:=context.WithCancel(context.Background())
 defer cancel()
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  _,_=w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"cancelled","type":"function","function":{"name":"safe_read","arguments":"{}"}}]}}]}`))
 }))
 defer srv.Close()
 reg:=core.NewRegistry()
 called:=0
 _=reg.Add(core.Command{ID:"safe.read",Path:"safe read",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){called++;cancel();return nil,context.Canceled}})
 a:=New(reg);a.URL=srv.URL;a.Key="test"
 a.Messages=[]message{{Role:"system",Content:"original"}}
 answer,err:=a.Ask(ctx,"diagnose")
 if err==nil||answer!=""{t.Fatalf("expected cancelled turn; answer=%q err=%v",answer,err)}
 if called!=1 {t.Fatalf("tool calls=%d",called)}
 if len(a.Messages)!=1||a.Messages[0].Content!="original"{t.Fatalf("history was not restored: %+v",a.Messages)}
}

func TestAgentInvalidArgumentsDoNotExecuteTool(t *testing.T) {
 turns,executed:=0,0
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  turns++
  var req struct {
   Tools []struct{Function struct{Parameters map[string]any `json:"parameters"`} `json:"function"`} `json:"tools"`
   Messages []struct{Role string `json:"role"`;Content any `json:"content"`} `json:"messages"`
  }
  if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{t.Error(err)}
  if turns==1 {
   if len(req.Tools)!=1||req.Tools[0].Function.Parameters["type"]!="object"{t.Errorf("bad advertised schema: %+v",req.Tools)}
   _,_=w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"bad","type":"function","function":{"name":"test_echo","arguments":"{\"message\":42}"}}]}}]}`))
   return
  }
  if len(req.Messages)==0 {t.Error("no tool result provided")}
  _,_=w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"handled"}}]}`))
 }))
 defer srv.Close()
 reg:=core.NewRegistry()
 schema:=map[string]any{"type":"object","properties":map[string]any{"message":map[string]any{"type":"string"}},"required":[]any{"message"},"additionalProperties":false}
 _=reg.Add(core.Command{ID:"test.echo",Path:"test echo",Risk:"read",InputSchema:schema,Handler:func(context.Context,map[string]any)(any,error){executed++;return "unexpected",nil}})
 a:=New(reg);a.URL=srv.URL;a.Key="test"
 answer,err:=a.Ask(context.Background(),"test")
 if err!=nil||answer!="handled"{t.Fatalf("answer=%q err=%v",answer,err)}
 if executed!=0 {t.Fatalf("invalid argument executed tool %d times",executed)}
 if len(a.Messages)<4{t.Fatalf("missing tool response: %+v",a.Messages)}
 found:=false
 for _,m:=range a.Messages {if m.Role=="tool" {text,_:=m.Content.(string);found=strings.Contains(text,"invalid input")}}
 if !found {t.Fatalf("schema error missing in tool response: %+v",a.Messages)}
}
