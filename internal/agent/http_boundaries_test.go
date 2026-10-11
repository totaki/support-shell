package agent

import (
 "context"
 "errors"
 "io"
 "net"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "example.com/support-shell/internal/core"
)

type timeoutOnceTransport struct{ calls int; next http.RoundTripper }
func (t *timeoutOnceTransport) RoundTrip(r *http.Request)(*http.Response,error){
 t.calls++
 if t.calls==2{return nil,&net.DNSError{IsTimeout:true,Err:"simulated timeout"}}
 return t.next.RoundTrip(r)
}

func TestAgentRetriesModelAfterToolWithoutRepeatingTool(t *testing.T){
 requests,executions:=0,0
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  requests++
  if requests==1{
   _,_=io.WriteString(w,`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call1","type":"function","function":{"name":"safe_read","arguments":"{}"}}]}}]}`)
   return
  }
  _,_=io.WriteString(w,`{"choices":[{"message":{"role":"assistant","content":"done"}}]}`)
 }))
 defer srv.Close()
 r:=core.NewRegistry()
 if err:=r.Add(core.Command{ID:"safe.read",Path:"safe read",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){executions++;return "ok",nil}});err!=nil{t.Fatal(err)}
 a:=New(r);a.URL=srv.URL;a.Key="test"
 transport:=&timeoutOnceTransport{next:http.DefaultTransport}
 a.Client=&http.Client{Transport:transport}
 got,err:=a.Ask(context.Background(),"diagnose")
 if err!=nil||got!="done"{t.Fatalf("answer=%q err=%v",got,err)}
 if transport.calls!=3||requests!=2||executions!=1{t.Fatalf("HTTP attempts=%d server requests=%d tool executions=%d",transport.calls,requests,executions)}
}

func TestAgentModelHTTPAndResponseErrors(t *testing.T){
 cases:=[]struct{name string; status int;body,contains string}{
  {"bad gateway",http.StatusBadGateway,"upstream unavailable","model HTTP 502"},
  {"unauthorized",http.StatusUnauthorized,"denied","model HTTP 401"},
  {"invalid JSON",http.StatusOK,"not json","invalid character"},
  {"empty choices",http.StatusOK,`{"choices":[]}`,"empty model response"},
 }
 for _,tc:=range cases{t.Run(tc.name,func(t *testing.T){
  srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(tc.status);_,_=io.WriteString(w,tc.body)}))
  defer srv.Close()
  a:=New(core.NewRegistry());a.URL=srv.URL;a.Key="test"
  _,err:=a.Ask(context.Background(),"diagnose")
  if err==nil||!strings.Contains(err.Error(),tc.contains){t.Fatalf("err=%v expected %q",err,tc.contains)}
 })}
}

func TestAgentCancelledBeforeRequest(t *testing.T){
 ctx,cancel:=context.WithCancel(context.Background());cancel()
 a:=New(core.NewRegistry());a.Key="test"
 a.Messages=[]message{{Role:"system",Content:"keep"}}
 _,err:=a.Ask(ctx,"request")
 if !errors.Is(err,context.Canceled){t.Fatalf("err=%v",err)}
 if len(a.Messages)!=1||a.Messages[0].Content!="keep"{t.Fatalf("history changed: %+v",a.Messages)}
}
