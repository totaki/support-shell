package pluginmanager

import (
 "context"
 "fmt"
 "sort"
 "sync"
 "example.com/support-shell/internal/core"
 "example.com/support-shell/pkg/pluginapi"
)

type Plugin struct {
 Name string `json:"name"`
 Version string `json:"version"`
 APIVersion string `json:"apiVersion"`
 Path string `json:"path"`
 Status string `json:"status"`
 Capabilities []string `json:"capabilities"`
 Commands []pluginapi.Command `json:"commands"`
}
type Manager struct { mu sync.RWMutex; items map[string]Plugin }
func New() *Manager { return &Manager{items:make(map[string]Plugin)} }
func (m *Manager) Add(path string, manifest pluginapi.Manifest) error {
 if err:=manifest.Validate();err!=nil{return err}
 m.mu.Lock(); defer m.mu.Unlock()
 if _, exists:=m.items[manifest.Name];exists{return fmt.Errorf("duplicate plugin %q",manifest.Name)}
 m.items[manifest.Name]=Plugin{Name:manifest.Name,Version:manifest.Version,APIVersion:manifest.APIVersion,Path:path,Status:"loaded",Capabilities:append([]string(nil),manifest.Capabilities...),Commands:append([]pluginapi.Command(nil),manifest.Commands...)}
 return nil
}
func (m *Manager) List() []Plugin {
 m.mu.RLock();defer m.mu.RUnlock()
 result:=make([]Plugin,0,len(m.items))
 for _,p:=range m.items {
 p.Capabilities=append([]string(nil),p.Capabilities...)
 p.Commands=append([]pluginapi.Command(nil),p.Commands...)
 result=append(result,p)
 }
 sort.Slice(result,func(i,j int)bool{return result[i].Name<result[j].Name})
 return result
}
func (m *Manager) Info(name string)(Plugin,bool){
 m.mu.RLock(); defer m.mu.RUnlock()
 p,ok:=m.items[name]
 p.Capabilities=append([]string(nil),p.Capabilities...)
 p.Commands=append([]pluginapi.Command(nil),p.Commands...)
 return p,ok
}
func (m *Manager) Register(r *core.Registry) error {
 if err:=r.Add(core.Command{ID:"plugins.list",Path:"plugins list",Description:"List loaded WASM plugins",Risk:"read",Handler:func(_ context.Context,_ map[string]any)(any,error){
  return map[string]any{"plugins":m.List()},nil
 }});err!=nil{return err}
 return r.Add(core.Command{ID:"plugins.info",Path:"plugins info",Description:"Show plugin metadata, capabilities and commands",Risk:"read",Args:[]string{"name"},Handler:func(_ context.Context,in map[string]any)(any,error){
  name,_:=in["name"].(string)
  p,ok:=m.Info(name)
  if !ok{return nil,fmt.Errorf("plugin %q not loaded",name)}
  return p,nil
 },Completer:func(_ context.Context,prefix string)[]string{
  result:=[]string{}
  for _,p:=range m.List(){if len(prefix)==0 || len(p.Name)>=len(prefix)&&p.Name[:len(prefix)]==prefix {result=append(result,"plugins info "+p.Name)}}
  return result
 }})
}
