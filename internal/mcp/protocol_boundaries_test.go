package mcp

import (
 "bytes"
 "context"
 "encoding/json"
 "strings"
 "testing"

 "example.com/support-shell/internal/core"
)

func TestMCPProtocolErrorsAndNotifications(t *testing.T) {
 registry:=core.NewRegistry()
 server:=Server{Registry:registry}
 input:=strings.Join([]string{
  "{bad JSON",
  `{"jsonrpc":"1.0","id":1,"method":"ping"}`,
  `{"jsonrpc":"2.0","id":2,"method":"unrecognized"}`,
  `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":42}}`,
  `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
  `{"jsonrpc":"2.0","id":4,"method":"ping"}`,
 },"\n")+"\n"
 var output bytes.Buffer
 if err:=server.Serve(context.Background(),strings.NewReader(input),&output);err!=nil{t.Fatal(err)}
 lines:=strings.Split(strings.TrimSpace(output.String()),"\n")
 if len(lines)!=5 {t.Fatalf("got %d responses: %s",len(lines),output.String())}
 codes:=[]float64{-32700,-32600,-32601,-32602,0}
 for i,line:=range lines {
  var resp map[string]any
  if err:=json.Unmarshal([]byte(line),&resp);err!=nil{t.Fatal(err)}
  if codes[i]!=0 {
   body,ok:=resp["error"].(map[string]any)
   if !ok||body["code"]!=codes[i]{t.Fatalf("line %d expected code %v: %s",i,codes[i],line)}
  } else if resp["error"]!=nil||resp["result"]==nil {
   t.Fatalf("ping failed: %s",line)
  }
 }
}

func TestMCPToolErrorsAndNilRegistry(t *testing.T) {
 if err:=(&Server{}).Serve(context.Background(),strings.NewReader(""),&bytes.Buffer{});err==nil{t.Fatal("nil Registry accepted")}
 r:=core.NewRegistry()
 _=r.Add(core.Command{ID:"test.fail",Path:"test fail",Risk:"read",Handler:func(context.Context,map[string]any)(any,error){return nil,context.DeadlineExceeded}})
 s:=Server{Registry:r}
 in:=strings.Join([]string{
  `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test.fail","arguments":{}}}`,
  `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"missing","arguments":{}}}`,
 },"\n")+"\n"
 var out bytes.Buffer
 if err:=s.Serve(context.Background(),strings.NewReader(in),&out);err!=nil{t.Fatal(err)}
 lines:=strings.Split(strings.TrimSpace(out.String()),"\n")
 if len(lines)!=2 {t.Fatal(out.String())}
 var tool map[string]any
 if err:=json.Unmarshal([]byte(lines[0]),&tool);err!=nil{t.Fatal(err)}
 if tool["result"].(map[string]any)["isError"]!=true{t.Fatalf("tool error missing: %s",lines[0])}
 var unknown map[string]any
 if err:=json.Unmarshal([]byte(lines[1]),&unknown);err!=nil{t.Fatal(err)}
 if unknown["error"]==nil{t.Fatalf("unknown tool accepted: %s",lines[1])}
}
