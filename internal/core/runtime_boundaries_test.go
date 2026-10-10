package core

import (
 "context"
 "testing"
)

func TestInputSchemaBoundaryCases(t *testing.T) {
 schema:=map[string]any{
  "type":"object",
  "properties":map[string]any{
   "name":map[string]any{"type":"string","enum":[]any{"alpha","beta"}},
   "count":map[string]any{"type":"integer"},
   "enabled":map[string]any{"type":"boolean"},
   "items":map[string]any{"type":"array"},
   "options":map[string]any{"type":"object"},
  },
  "required":[]string{"name"},
  "additionalProperties":false,
 }
 cases:=[]struct{name string;value map[string]any;valid bool}{
  {"valid",map[string]any{"name":"alpha","count":float64(2),"enabled":true,"items":[]any{},"options":map[string]any{}},true},
  {"nil input",nil,false},
  {"missing required",map[string]any{"count":1},false},
  {"unknown field",map[string]any{"name":"alpha","unexpected":true},false},
  {"wrong enum",map[string]any{"name":"gamma"},false},
  {"fractional integer",map[string]any{"name":"alpha","count":2.5},false},
  {"invalid boolean",map[string]any{"name":"alpha","enabled":"true"},false},
  {"invalid array",map[string]any{"name":"alpha","items":"items"},false},
  {"invalid object",map[string]any{"name":"alpha","options":[]any{}},false},
  {"beta",map[string]any{"name":"beta"},true},
 }
 for _,tc:=range cases {
  t.Run(tc.name,func(t *testing.T){
   err:=validateInput(schema,tc.value)
   if (err==nil)!=tc.valid {t.Fatalf("valid=%v, err=%v",tc.valid,err)}
  })
 }
 if err:=validateInput(map[string]any{"type":"array"},map[string]any{});err==nil{t.Fatal("non-object schema accepted")}
 if err:=validateInput(map[string]any{"type":"object","required":[]any{17}},map[string]any{});err==nil{t.Fatal("malformed required accepted")}
 if err:=validateInput(nil,nil);err!=nil{t.Fatal(err)}
}

func TestRuntimeApprovalAndUnknownCommands(t *testing.T) {
 r:=NewRegistry()
 calls:=0
 handler:=func(context.Context,map[string]any)(any,error){calls++;return "ok",nil}
 if err:=r.Add(Command{ID:"write.operation",Path:"write operation",Risk:"write",Handler:handler});err!=nil{t.Fatal(err)}
 if _,err:=r.Execute(context.Background(),"write.operation",nil);err==nil{t.Fatal("write executed without approval")}
 if calls!=0{t.Fatal("write handler ran without approval")}
 if _,err:=r.ExecuteApproved(context.Background(),"write.operation",nil);err!=nil{t.Fatal(err)}
 if calls!=1{t.Fatalf("calls=%d",calls)}
 if _,err:=r.Execute(context.Background(),"missing",nil);err==nil{t.Fatal("unknown ID accepted")}
 if err:=r.Add(Command{ID:"write.operation",Path:"another path",Risk:"read",Handler:handler});err==nil{t.Fatal("duplicate ID accepted")}
 if err:=r.Add(Command{ID:"another.operation",Path:"write operation",Risk:"read",Handler:handler});err==nil{t.Fatal("duplicate path accepted")}
}
