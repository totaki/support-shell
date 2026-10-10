package mcp

import (
 "bytes"
 "context"
 "encoding/json"
 "strings"
 "testing"

 "example.com/support-shell/internal/core"
)

func TestMCPClientHandshakeAndProtocolSemantics(t *testing.T) {
 registry:=core.NewRegistry()
 if err:=registry.Add(core.Command{ID:"sample.read",Path:"sample read",Risk:"read",
  Handler:func(context.Context,map[string]any)(any,error){return map[string]any{"ok":true},nil},
 });err!=nil{t.Fatal(err)}
 srv:=Server{Registry:registry}
 input:=strings.Join([]string{
  `{"jsonrpc":"2.0","id":"init-1","method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"compat","version":"1"},"capabilities":{}}}`,
  `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
  `{"jsonrpc":"2.0","id":7,"method":"ping"}`,
  `{"jsonrpc":"2.0","id":"list","method":"tools/list","params":{}}`,
  `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"sample.read","arguments":{}}}`,
 },"\n")+"\n"
 var buf bytes.Buffer
 if err:=srv.Serve(context.Background(),strings.NewReader(input),&buf);err!=nil{t.Fatal(err)}
 lines:=strings.Split(strings.TrimSpace(buf.String()),"\n")
 if len(lines)!=4{t.Fatalf("notifications must not generate responses: %s",buf.String())}
 for i,line:=range lines{
  var frame map[string]any
  if err:=json.Unmarshal([]byte(line),&frame);err!=nil{t.Fatal(err)}
  if frame["jsonrpc"]!="2.0"||frame["error"]!=nil{t.Fatalf("frame %d: %v",i,frame)}
  if frame["result"]==nil{t.Fatalf("missing result: %v",frame)}
 }
 var init map[string]any
 _=json.Unmarshal([]byte(lines[0]),&init)
 if init["id"]!="init-1"{t.Fatalf("string ID not preserved: %v",init)}
 if init["result"].(map[string]any)["protocolVersion"]!=protocolVersion{t.Fatal(init)}
 var list map[string]any
 _=json.Unmarshal([]byte(lines[2]),&list)
 tools:=list["result"].(map[string]any)["tools"].([]any)
 if len(tools)!=1{t.Fatalf("advertised tools: %v",tools)}
 advertised:=tools[0].(map[string]any)
 if advertised["name"]!="sample.read"{t.Fatal(advertised)}
 if advertised["annotations"].(map[string]any)["readOnlyHint"]!=true{t.Fatal(advertised)}
 var call map[string]any
 _=json.Unmarshal([]byte(lines[3]),&call)
 if call["result"].(map[string]any)["structuredContent"].(map[string]any)["ok"]!=true{t.Fatal(call)}
}
