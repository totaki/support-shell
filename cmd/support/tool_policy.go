package main

import (
 "fmt"
 "os"
 "strings"

 "example.com/support-shell/internal/core"
 "gopkg.in/yaml.v3"
)

type policyFile struct {
 Deny map[string][]string `yaml:"deny"`
}

// configureToolPolicy loads an optional YAML file and applies environment
// denials on top. Both sources only remove access, never grant it.
func configureToolPolicy(r *core.Registry) error {
 deny:=map[core.Surface]map[string]bool{}
 names:=map[core.Surface]string{
  core.SurfaceCLI:"SUPPORT_DENY_CLI",
  core.SurfaceAgent:"SUPPORT_DENY_AGENT",
  core.SurfaceMCP:"SUPPORT_DENY_MCP",
 }
 for surface:=range names {deny[surface]=map[string]bool{}}
 if path:=strings.TrimSpace(os.Getenv("SUPPORT_POLICY_FILE"));path!="" {
  raw,err:=os.ReadFile(path)
  if err!=nil{return fmt.Errorf("read policy file: %w",err)}
  var config policyFile
  if err:=yaml.Unmarshal(raw,&config);err!=nil{return fmt.Errorf("invalid tool policy YAML: %w",err)}
  for name,ids:=range config.Deny {
   surface:=core.Surface(name)
   if _,ok:=names[surface];!ok{return fmt.Errorf("unknown policy interface %q",name)}
   for _,id:=range ids {
    id=strings.TrimSpace(id)
    if id=="" {return fmt.Errorf("empty command ID for %s",name)}
    deny[surface][id]=true
   }
  }
 }
 for surface,key:=range names {
  for _,raw:=range strings.Split(os.Getenv(key),","){
   if id:=strings.TrimSpace(raw);id!="" {deny[surface][id]=true}
  }
 }
 r.SetToolPolicy(core.ToolPolicy{Deny:deny})
 return nil
}
