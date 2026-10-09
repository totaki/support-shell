package core

import (
 "fmt"
 "math"
)

// validateInput checks the supported subset of JSON Schema for command arguments.
func validateInput(schema map[string]any, input map[string]any) error {
 if schema==nil {return nil}
 if schema["type"]!="object" {return fmt.Errorf("input schema must be an object")}
 if input==nil {input=map[string]any{}}
 required:=[]string{}
 switch v:=schema["required"].(type) {
 case []string: required=append(required,v...)
 case []any:
  for _,raw:=range v {name,ok:=raw.(string);if !ok{return fmt.Errorf("invalid required field")};required=append(required,name)}
 }
 for _,name:=range required {if _,ok:=input[name];!ok{return fmt.Errorf("missing required argument %q",name)}}
 properties,_:=schema["properties"].(map[string]any)
 for name,value:=range input {
  raw,exists:=properties[name]
  if !exists {
   if schema["additionalProperties"]==false {return fmt.Errorf("unknown argument %q",name)}
   continue
  }
  rule,ok:=raw.(map[string]any);if !ok {continue}
  typ,_:=rule["type"].(string)
  valid:=true
  switch typ {
  case "string":_,valid=value.(string)
  case "boolean":_,valid=value.(bool)
  case "object":_,valid=value.(map[string]any)
  case "array":_,valid=value.([]any)
  case "number":valid=numeric(value)
  case "integer":valid=integer(value)
  }
  if !valid{return fmt.Errorf("argument %q must be %s",name,typ)}
  if options,ok:=rule["enum"].([]any);ok {
   found:=false
   for _,item:=range options {if fmt.Sprint(item)==fmt.Sprint(value){found=true;break}}
   if !found{return fmt.Errorf("argument %q is outside enum",name)}
  }
 }
 return nil
}
func numeric(v any)bool {
 switch v.(type){case int,int32,int64,uint,float32,float64:return true}
 return false
}
func integer(v any)bool {
 switch n:=v.(type){
 case int,int32,int64,uint:return true
 case float64:return !math.IsInf(n,0)&&!math.IsNaN(n)&&n==math.Trunc(n)
 }
 return false
}
