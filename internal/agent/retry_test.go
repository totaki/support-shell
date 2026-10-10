package agent

import (
 "context"
 "errors"
 "net/http"
 "net/http/httptest"
 "testing"

 "example.com/support-shell/internal/core"
)

type failOnceTransport struct {
 underlying http.RoundTripper
 calls int
}
func (t *failOnceTransport) RoundTrip(req *http.Request)(*http.Response,error){
 t.calls++
 if t.calls==2 {return nil,context.DeadlineExceeded}
 return t.underlying.RoundTrip(req)
}

func TestModelRetryNeverReexecutesTool(t *testing.T) {
 executed:=0
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,_ *http.Request){
  if executed==0{
   _,_=w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"t1","type":"function","function":{"name":"safe_read","arguments":"{}"}}]}}]}`))
  }else{
   _,_=w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"completed"}}]}`))
  }
 }))
 defer server.Close()
 registry:=core.NewRegistry()
 if err:=registry.Add(core.Command{ID:"safe.read",Path:"safe read",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){executed++;return "ok",nil}});err!=nil{t.Fatal(err)}
 agent:=New(registry);agent.URL=server.URL;agent.Key="test"
 transport:= &failOnceTransport{underlying:http.DefaultTransport}
 agent.Client=&http.Client{Transport:transport}
 answer,err:=agent.Ask(context.Background(),"check")
 if err!=nil||answer!="completed"{t.Fatalf("answer=%q err=%v",answer,err)}
 if transport.calls!=3{t.Fatalf("HTTP calls=%d, expected one timed-out retry",transport.calls)}
 if executed!=1{t.Fatalf("tool executed %d times",executed)}
}

func TestTimeoutDetection(t *testing.T){
 if !isTimeout(context.DeadlineExceeded){t.Fatal("deadline is a timeout")}
 if isTimeout(errors.New("other")){t.Fatal("plain error classified as timeout")}
}
