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

func TestMCPUsesRegistryValidation(t *testing.T) {
 r:=core.NewRegistry()
 executions:=0
 err:=r.Add(core.Command{
  ID:"test.validated",Path:"test validated",Risk:"read",Description:"Validated test tool",
  InputSchema:map[string]any{"type":"object","properties":map[string]any{"name":map[string]any{"type":"string"}},"required":[]any{"name"},"additionalProperties":false},
  Handler:func(_ context.Context,in map[string]any)(any,error){executions++;return map[string]any{"value":in["name"]},nil},
 })
 if err!=nil {t.Fatal(err)}
 server:=Server{Registry:r}
 input:=strings.Join([]string{
  `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
  `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"test.validated","arguments":{"name":12}}}`,
  `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"test.validated","arguments":{"name":"hello"}}}`,
 },"\n")+"\n"
 var output bytes.Buffer
 if err:=server.Serve(context.Background(),strings.NewReader(input),&output);err!=nil{t.Fatal(err)}
 lines:=strings.Split(strings.TrimSpace(output.String()),"\n")
 if len(lines)!=3{t.Fatalf("response count: %s",output.String())}
 var advertised map[string]any
 if err:=json.Unmarshal([]byte(lines[0]),&advertised);err!=nil{t.Fatal(err)}
 tools:=advertised["result"].(map[string]any)["tools"].([]any)
 schema:=tools[0].(map[string]any)["inputSchema"].(map[string]any)
 if schema["type"]!="object"{t.Fatalf("schema not advertised: %v",schema)}
 var invalid map[string]any
 _=json.Unmarshal([]byte(lines[1]),&invalid)
 if invalid["result"].(map[string]any)["isError"]!=true{t.Fatalf("expected tool error: %s",lines[1])}
 if executions!=1{t.Fatalf("invalid call executed handler; executions=%d",executions)}
 var valid map[string]any
 _=json.Unmarshal([]byte(lines[2]),&valid)
 data:=valid["result"].(map[string]any)["structuredContent"].(map[string]any)
 if data["value"]!="hello"{t.Fatalf("unexpected structured result: %v",data)}
}
