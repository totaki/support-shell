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
