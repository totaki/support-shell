package core

import (
 "context"
 "testing"
)

func TestSurfacePolicy(t *testing.T) {
 r:=NewRegistry()
 calls:=0
 for _,item:=range []struct{id,risk string}{{"test.read","read"},{"test.write","write"},{"demo.cluster.test","read"}} {
  id:=item.id
  err:=r.Add(Command{ID:id,Path:id,Risk:item.risk,Handler:func(context.Context,map[string]any)(any,error){calls++;return "ok",nil}})
  if err!=nil{t.Fatal(err)}
 }
 r.SetToolPolicy(ToolPolicy{Deny:map[Surface]map[string]bool{SurfaceMCP:{"test.read":true},SurfaceAgent:{"test.read":true}}})
 for _,surface:=range []Surface{SurfaceAgent,SurfaceMCP}{
  if len(r.ListFor(surface))!=0{t.Fatalf("unexpected %s exposure",surface)}
  if _,err:=r.ExecuteFor(context.Background(),surface,"test.read",nil);err==nil{t.Fatalf("%s bypassed denial",surface)}
 }
 if _,err:=r.ExecuteFor(context.Background(),SurfaceCLI,"test.read",nil);err!=nil{t.Fatal(err)}
 if calls!=1{t.Fatalf("handler was called %d times",calls)}
 if _,err:=r.ExecuteFor(context.Background(),SurfaceCLI,"test.write",nil);err==nil{t.Fatal("write bypassed approval")}
}

func TestToolPolicyReasonsAndIsolation(t *testing.T) {
 r:=NewRegistry()
 calls:=0
 add:=func(id,risk string){
  t.Helper()
  if err:=r.Add(Command{ID:id,Path:id,Risk:risk,Handler:func(context.Context,map[string]any)(any,error){calls++;return "ok",nil}});err!=nil{t.Fatal(err)}
 }
 add("safe.read","read")
 add("restricted.read","read")
 add("unsafe.write","write")
 add("demo.cluster.fake","read")
 r.SetToolPolicy(ToolPolicy{Deny:map[Surface]map[string]bool{
  SurfaceMCP:{"restricted.read":true},
  SurfaceCLI:{"safe.read":true},
 }})
 checks:=[]struct{surface Surface;id,reason string}{
  {SurfaceMCP,"safe.read",""},
  {SurfaceMCP,"restricted.read","denied by policy"},
  {SurfaceMCP,"unsafe.write","non-read command"},
  {SurfaceMCP,"demo.cluster.fake","demo cluster tool"},
  {SurfaceAgent,"restricted.read",""},
  {SurfaceCLI,"safe.read","denied by policy"},
  {SurfaceCLI,"unsafe.write",""},
  {Surface("unrecognized"),"safe.read","unknown interface"},
 }
 for _,tc:=range checks {
  command,_,ok:=r.Resolve(tc.id)
  if !ok{t.Fatal(tc.id)}
  if got:=r.AccessReason(tc.surface,command);got!=tc.reason{t.Errorf("%s/%s reason=%q want %q",tc.surface,tc.id,got,tc.reason)}
  _,err:=r.ExecuteFor(context.Background(),tc.surface,tc.id,nil)
  shouldSucceed:=tc.reason=="" && tc.id!="unsafe.write"
  if (err==nil)!=shouldSucceed{t.Errorf("%s/%s err=%v wantSuccess=%v",tc.surface,tc.id,err,shouldSucceed)}
 }
 if calls!=2{t.Fatalf("only allowed read handlers should run, calls=%d",calls)}
 if _,err:=r.ExecuteFor(context.Background(),SurfaceMCP,"absent",nil);err==nil{t.Fatal("unknown command accepted")}
}
