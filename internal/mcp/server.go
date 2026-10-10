// Package mcp exposes the same read-only Registry tools via JSON-RPC 2.0 over stdio.
package mcp

import (
 "bufio"
 "context"
 "encoding/json"
 "fmt"
 "io"

 "example.com/support-shell/internal/core"
)

const protocolVersion = "2025-03-26"

type request struct {
 JSONRPC string          `json:"jsonrpc"`
 ID      json.RawMessage `json:"id,omitempty"`
 Method  string          `json:"method"`
 Params  json.RawMessage `json:"params,omitempty"`
}
type rpcError struct {
 Code int `json:"code"`
 Message string `json:"message"`
}
type response struct {
 JSONRPC string `json:"jsonrpc"`
 ID json.RawMessage `json:"id"`
 Result any `json:"result,omitempty"`
 Error *rpcError `json:"error,omitempty"`
}
type Server struct { Registry *core.Registry }

func (s *Server) listTools() []map[string]any {
 out:=[]map[string]any{}
 for _,c:=range s.Registry.ListFor(core.SurfaceMCP){
  schema:=c.InputSchema
  if schema==nil {
   properties:=map[string]any{}
   for _,arg:=range c.Args {properties[arg]=map[string]any{"type":"string"}}
   required:=append([]string{},c.Args...)
   schema=map[string]any{"type":"object","properties":properties,"required":required,"additionalProperties":false}
  }
  entry:=map[string]any{"name":c.ID,"title":c.Path,"description":c.Description,"inputSchema":schema,"annotations":map[string]any{"readOnlyHint":true}}
  if c.OutputSchema!=nil {entry["outputSchema"]=c.OutputSchema}
  out=append(out,entry)
 }
 return out
}
func (s *Server) dispatch(ctx context.Context, req request) response {
 answer:=response{JSONRPC:"2.0",ID:req.ID}
 fail:=func(code int,msg string)response{answer.Error=&rpcError{Code:code,Message:msg};return answer}
 switch req.Method {
 case "initialize":
  answer.Result=map[string]any{"protocolVersion":protocolVersion,"capabilities":map[string]any{"tools":map[string]any{"listChanged":false}},"serverInfo":map[string]any{"name":"support-shell","version":"0.1.0"}}
 case "ping":
  answer.Result=map[string]any{}
 case "tools/list":
  answer.Result=map[string]any{"tools":s.listTools()}
 case "tools/call":
  var params struct {
   Name string `json:"name"`
   Arguments map[string]any `json:"arguments"`
  }
  if err:=json.Unmarshal(req.Params,&params);err!=nil||params.Name=="" {return fail(-32602,"invalid tool arguments")}
  allowed:=false
  for _,tool:=range s.listTools(){if tool["name"]==params.Name {allowed=true;break}}
  if !allowed {return fail(-32602,"unknown or unavailable tool")}
  if params.Arguments==nil {params.Arguments=map[string]any{}}
  data,err:=s.Registry.ExecuteFor(ctx,core.SurfaceMCP,params.Name,params.Arguments)
  if err!=nil {answer.Result=map[string]any{"isError":true,"content":[]map[string]any{{"type":"text","text":err.Error()}}};break}
  result:=map[string]any{"content":[]map[string]any{{"type":"text","text":core.JSON(data)}}}
  if _,err:=json.Marshal(data);err==nil {result["structuredContent"]=data}
  answer.Result=result
 default:
  return fail(-32601,"method not found")
 }
 return answer
}

// Serve accepts newline-delimited JSON-RPC messages; stdout is reserved
// for protocol frames, and notifications never receive responses.
func (s *Server) Serve(ctx context.Context, input io.Reader, output io.Writer) error {
 if s.Registry==nil {return fmt.Errorf("nil registry")}
 scanner:=bufio.NewScanner(input)
 scanner.Buffer(make([]byte,4096),4<<20)
 writer:=bufio.NewWriter(output)
 for scanner.Scan(){
  if err:=ctx.Err();err!=nil{return err}
  var req request
  if err:=json.Unmarshal(scanner.Bytes(),&req);err!=nil {
   if err:=write(writer,response{JSONRPC:"2.0",ID:json.RawMessage("null"),Error:&rpcError{Code:-32700,Message:"parse error"}});err!=nil{return err}
   continue
  }
  if req.JSONRPC!="2.0" || req.Method=="" {
   if err:=write(writer,response{JSONRPC:"2.0",ID:json.RawMessage("null"),Error:&rpcError{Code:-32600,Message:"invalid request"}});err!=nil{return err}
   continue
  }
  if len(req.ID)==0 {continue}
  if err:=write(writer,s.dispatch(ctx,req));err!=nil{return err}
 }
 return scanner.Err()
}
func write(w *bufio.Writer,resp response) error {
 data,err:=json.Marshal(resp);if err!=nil{return err}
 if _,err=w.Write(append(data,'\n'));err!=nil{return err}
 return w.Flush()
}
