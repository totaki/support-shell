// Package provider defines infrastructure capabilities independently of CLI and WASM.
package provider

import (
 "context"
 "errors"
 "fmt"
 "sort"
 "strings"
 "sync"

 "example.com/support-shell/internal/core"
)

type Operation struct {
 ID string
 Path string
 Description string
 Risk string
 Args []string
 InputSchema map[string]any
 OutputSchema map[string]any
 Examples []string
 Handler func(context.Context,map[string]any)(any,error)
 Completer func(context.Context,string)[]string
}

type Provider interface {
 ID() string
 Operations() []Operation
}

type Catalog struct {
 id string
 operations []Operation
}

func NewCatalog(id string)*Catalog{return &Catalog{id:id}}
func(c *Catalog) ID()string{return c.id}
func(c *Catalog) Operations()[]Operation{return append([]Operation(nil),c.operations...)}
func(c *Catalog) Add(o Operation)error{
 if o.ID==""||o.Path==""||o.Handler==nil{return errors.New("invalid provider operation")}
 for _,current:=range c.operations {
  if current.ID==o.ID||current.Path==o.Path{return fmt.Errorf("duplicate provider operation %s",o.ID)}
 }
 c.operations=append(c.operations,o)
 return nil
}

type Registry struct {
 mu sync.RWMutex
 providers map[string]Provider
 operations map[string]Operation
}

func NewRegistry()*Registry{
 return &Registry{providers:map[string]Provider{},operations:map[string]Operation{}}
}

func (r *Registry) Register(p Provider)error{
 if p==nil||p.ID()==""||strings.Contains(p.ID(),"."){return errors.New("invalid provider ID")}
 operations:=p.Operations()
 if len(operations)==0{return fmt.Errorf("provider %s exposes no operations",p.ID())}
 r.mu.Lock()
 defer r.mu.Unlock()
 if _,exists:=r.providers[p.ID()];exists{return fmt.Errorf("provider %s already registered",p.ID())}
 seen:=map[string]bool{}
 paths:=map[string]bool{}
 for _,op:=range operations {
  if !strings.HasPrefix(op.ID,"provider."+p.ID()+".")||op.Path==""||op.Handler==nil{return fmt.Errorf("invalid operation %s for provider %s",op.ID,p.ID())}
  if op.Risk!="read"&&op.Risk!="write"&&op.Risk!="dangerous"{return fmt.Errorf("invalid risk for %s",op.ID)}
  if seen[op.ID]||paths[op.Path]{return fmt.Errorf("duplicate operation in %s",p.ID())}
  if _,exists:=r.operations[op.ID];exists{return fmt.Errorf("operation %s already registered",op.ID)}
  seen[op.ID]=true;paths[op.Path]=true
 }
 for _,op:=range operations{r.operations[op.ID]=op}
 r.providers[p.ID()]=p
 return nil
}

func(r *Registry) IDs()[]string{
 r.mu.RLock();defer r.mu.RUnlock()
 ids:=make([]string,0,len(r.providers))
 for id:=range r.providers{ids=append(ids,id)}
 sort.Strings(ids)
 return ids
}
func(r *Registry) Operations()[]Operation{
 r.mu.RLock();defer r.mu.RUnlock()
 out:=make([]Operation,0,len(r.operations))
 for _,op:=range r.operations{out=append(out,op)}
 sort.Slice(out,func(i,j int)bool{return out[i].ID<out[j].ID})
 return out
}
func(r *Registry) Readable(id string)bool{
 r.mu.RLock();defer r.mu.RUnlock()
 op,ok:=r.operations[id]
 return ok&&op.Risk=="read"
}
func(r *Registry) ExecuteRead(ctx context.Context,id string,in map[string]any)(any,error){
 r.mu.RLock();op,ok:=r.operations[id];r.mu.RUnlock()
 if !ok{return nil,fmt.Errorf("unknown provider capability %s",id)}
 if op.Risk!="read"{return nil,fmt.Errorf("capability %s requires operator approval",id)}
 if err:=ctx.Err();err!=nil{return nil,err}
 if err:=core.ValidateInput(op.InputSchema,in);err!=nil{return nil,fmt.Errorf("invalid capability input: %w",err)}
 return op.Handler(ctx,in)
}

// BindTools publishes provider operations to CLI, MCP and Agent through the
// regular Core policy and approval checks. The plugin host never calls this layer.
func(r *Registry) BindTools(to *core.Registry)error{
 if err:=to.Add(core.Command{
  ID:"providers.list",Path:"providers",Risk:"read",
  Description:"List registered infrastructure providers",
  Handler:func(context.Context,map[string]any)(any,error){
   result:=[]map[string]any{}
   for _,id:=range r.IDs(){
    count:=0
    for _,operation:=range r.Operations(){if strings.HasPrefix(operation.ID,"provider."+id+"."){count++}}
    result=append(result,map[string]any{"id":id,"operations":count})
   }
   return result,nil
  },
 });err!=nil{return err}
 if err:=to.Add(core.Command{
  ID:"providers.capabilities",Path:"providers capabilities",Risk:"read",
  Description:"Show capabilities exported by infrastructure providers",
  Handler:func(context.Context,map[string]any)(any,error){
   result:=[]map[string]any{}
   for _,o:=range r.Operations(){result=append(result,map[string]any{"id":o.ID,"path":o.Path,"risk":o.Risk})}
   return result,nil
  },
 });err!=nil{return err}
 for _,op:=range r.Operations(){
  o:=op
  command:=core.Command{ID:o.ID,Path:o.Path,Description:o.Description,Risk:o.Risk,Args:o.Args,InputSchema:o.InputSchema,OutputSchema:o.OutputSchema,Examples:o.Examples,Completer:o.Completer,Handler:o.Handler}
  if err:=to.Add(command);err!=nil{return err}
 }
 return nil
}
