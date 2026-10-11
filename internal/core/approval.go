package core

import (
 "context"
 "crypto/rand"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "sync"
 "time"
)

type Approval struct {
 ID string
 CommandID string
 Input map[string]any
 Scope string
 ExpiresAt time.Time
}
type ApprovalManager struct {
 mu sync.Mutex
 pending map[string]Approval
}
func NewApprovalManager()*ApprovalManager{return &ApprovalManager{pending:map[string]Approval{}}}
func(m *ApprovalManager) Request(ctx context.Context,r *Registry,id string,input map[string]any,scope string)(Approval,error){
 if err:=ctx.Err();err!=nil{return Approval{},err}
 found:=false
 for _,c:=range r.List(){
  if c.ID!=id{continue}
  if c.Risk=="read"{return Approval{},errors.New("read commands do not require approval")}
  if !r.Allowed(SurfaceCLI,c){return Approval{},errors.New("command denied by CLI policy")}
  if err:=validateInput(c.InputSchema,input);err!=nil{return Approval{},err}
  found=true;break
 }
 if !found{return Approval{},fmt.Errorf("unknown command ID %s",id)}
 raw,err:=json.Marshal(input);if err!=nil{return Approval{},err}
 var copied map[string]any
 if err:=json.Unmarshal(raw,&copied);err!=nil{return Approval{},err}
 token:=make([]byte,16)
 if _,err:=rand.Read(token);err!=nil{return Approval{},err}
 request:=Approval{ID:hex.EncodeToString(token),CommandID:id,Input:copied,Scope:scope,ExpiresAt:time.Now().Add(5*time.Minute)}
 m.mu.Lock()
 if m.pending==nil{m.pending=map[string]Approval{}}
 m.pending[request.ID]=request
 m.mu.Unlock()
 return request,nil
}
func(m *ApprovalManager) Deny(id string){m.mu.Lock();delete(m.pending,id);m.mu.Unlock()}
func(m *ApprovalManager) Execute(ctx context.Context,r *Registry,id,scope string)(any,error){
 m.mu.Lock();request,ok:=m.pending[id];delete(m.pending,id);m.mu.Unlock()
 if !ok{return nil,errors.New("approval not found or already used")}
 if err:=ctx.Err();err!=nil{return nil,err}
 if time.Now().After(request.ExpiresAt){return nil,errors.New("approval expired")}
 if request.Scope!=scope{return nil,errors.New("approval scope changed")}
 for _,c:=range r.List(){
  if c.ID!=request.CommandID{continue}
  if c.Risk=="read"||!r.Allowed(SurfaceCLI,c){return nil,errors.New("command no longer allowed")}
  return r.ExecuteApproved(ctx,request.CommandID,request.Input)
 }
 return nil,errors.New("command no longer registered")
}
