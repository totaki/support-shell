package core

import (
 "context"
 "strings"
 "testing"
)

func TestRegistryRejectsDuplicateIDs(t *testing.T) {
 r:=NewRegistry()
 handler:=func(context.Context,map[string]any)(any,error){return "ok",nil}
 if err:=r.Add(Command{ID:"duplicate",Path:"one",Risk:"read",Handler:handler});err!=nil{t.Fatal(err)}
 if err:=r.Add(Command{ID:"duplicate",Path:"two",Risk:"read",Handler:handler});err==nil{t.Fatal("duplicate ID accepted")}
}

func TestRegistryInputValidation(t *testing.T) {
 r:=NewRegistry()
 calls:=0
 schema:=map[string]any{"type":"object","properties":map[string]any{"namespace":map[string]any{"type":"string"},"limit":map[string]any{"type":"integer"}},"required":[]any{"namespace"},"additionalProperties":false}
 err:=r.Add(Command{ID:"test.validate",Path:"test validate",Risk:"read",InputSchema:schema,Handler:func(_ context.Context,in map[string]any)(any,error){calls++;return in,nil}})
 if err!=nil{t.Fatal(err)}
 cases:=[]struct{name string;input map[string]any;wantError bool}{
  {"valid",map[string]any{"namespace":"default","limit":float64(5)},false},
  {"missing",map[string]any{},true},
  {"wrong type",map[string]any{"namespace":3},true},
  {"unknown",map[string]any{"namespace":"default","extra":true},true},
 }
 for _,tc:=range cases {
  t.Run(tc.name,func(t *testing.T){
   _,err:=r.Execute(context.Background(),"test.validate",tc.input)
   if (err!=nil)!=tc.wantError{t.Fatalf("err=%v expected error=%v",err,tc.wantError)}
  })
 }
 if calls!=1{t.Fatalf("handler executed %d times",calls)}
}

func TestValidateSchemaFromGoArgs(t *testing.T) {
 err:=validateInput(map[string]any{"type":"object","required":[]string{"name"}} ,map[string]any{})
 if err==nil||!strings.Contains(err.Error(),"name"){t.Fatalf("unexpected error: %v",err)}
}
