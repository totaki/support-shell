package core

import (
 "context"
 "testing"
)
func TestApprovalSingleUseAndScope(t *testing.T){
 r:=NewRegistry();calls:=0;var target any
 err:=r.Add(Command{ID:"change",Path:"change",Risk:"write",Handler:func(_ context.Context,in map[string]any)(any,error){calls++;target=in["target"];return "ok",nil}})
 if err!=nil{t.Fatal(err)}
 m:=NewApprovalManager()
 in:=map[string]any{"target":"before"}
 req,err:=m.Request(context.Background(),r,"change",in,"cluster-a");if err!=nil{t.Fatal(err)}
 in["target"]="after"
 if _,err:=m.Execute(context.Background(),r,req.ID,"cluster-b");err==nil{t.Fatal("scope mismatch permitted")}
 if _,err:=m.Execute(context.Background(),r,req.ID,"cluster-a");err==nil{t.Fatal("consumed token reused")}
 req,err=m.Request(context.Background(),r,"change",map[string]any{"target":"original"},"cluster-a");if err!=nil{t.Fatal(err)}
 if _,err:=m.Execute(context.Background(),r,req.ID,"cluster-a");err!=nil{t.Fatal(err)}
 if calls!=1||target!="original"{t.Fatalf("calls=%d target=%v",calls,target)}
 if _,err:=m.Execute(context.Background(),r,req.ID,"cluster-a");err==nil{t.Fatal("replay permitted")}
}
func TestApprovalDenyAndPolicy(t *testing.T){
 r:=NewRegistry();calls:=0
 _=r.Add(Command{ID:"write",Path:"write",Risk:"write",Handler:func(context.Context,map[string]any)(any,error){calls++;return nil,nil}})
 m:=NewApprovalManager()
 if _,err:=m.Request(context.Background(),r,"unknown",nil,"");err==nil{t.Fatal("unknown command approved")}
 request,err:=m.Request(context.Background(),r,"write",nil,"");if err!=nil{t.Fatal(err)}
 m.Deny(request.ID)
 if _,err:=m.Execute(context.Background(),r,request.ID,"");err==nil{t.Fatal("denied token succeeded")}
 request,err=m.Request(context.Background(),r,"write",nil,"");if err!=nil{t.Fatal(err)}
 r.SetToolPolicy(ToolPolicy{Deny:map[Surface]map[string]bool{SurfaceCLI:{"write":true}}})
 if _,err:=m.Execute(context.Background(),r,request.ID,"");err==nil{t.Fatal("policy bypassed")}
 if calls!=0{t.Fatalf("unexpected handler calls: %d",calls)}
}
