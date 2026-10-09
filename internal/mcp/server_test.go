package mcp

import (
 "bytes"
 "context"
 "encoding/json"
 "strings"
 "testing"
 "example.com/support-shell/internal/core"
)
func TestMCPRoundtrip(t *testing.T) {
 r:=core.NewRegistry()
 if err:=r.Add(core.Command{ID:"test.hello",Path:"test hello",Risk:"read",Description:"Say hello",Handler:func(_ context.Context,in map[string]any)(any,error){return map[string]any{"hello":in["name"]},nil}});err!=nil{t.Fatal(err)}
 if err:=r.Add(core.Command{ID:"test.mutate",Path:"test mutate",Risk:"write",Handler:func(_ context.Context,_ map[string]any)(any,error){return "oops",nil}});err!=nil{t.Fatal(err)}
 s:=Server{Registry:r}
 input:=strings.Join([]string{
  `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
  `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
  `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
  `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"test.hello","arguments":{"name":"world"}}}`,
  `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"test.mutate"}}`,
 },"\n")+"\n"
 var buf bytes.Buffer
 if err:=s.Serve(context.Background(),strings.NewReader(input),&buf);err!=nil{t.Fatal(err)}
 lines:=strings.Split(strings.TrimSpace(buf.String()),"\n")
 if len(lines)!=4{t.Fatalf("got %d responses: %s",len(lines),buf.String())}
 var list map[string]any
 if err:=json.Unmarshal([]byte(lines[1]),&list);err!=nil{t.Fatal(err)}
 tools:=list["result"].(map[string]any)["tools"].([]any)
 if len(tools)!=1||tools[0].(map[string]any)["name"]!="test.hello"{t.Fatalf("unexpected tools: %v",tools)}
 var call map[string]any
 if err:=json.Unmarshal([]byte(lines[2]),&call);err!=nil{t.Fatal(err)}
 if call["result"].(map[string]any)["isError"]==true {t.Fatal(call)}
 var denied map[string]any
 _=json.Unmarshal([]byte(lines[3]),&denied)
 if denied["error"]==nil{t.Fatalf("write tool was not blocked: %s",lines[3])}
}
