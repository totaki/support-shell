package provider

import (
 "context"
 "errors"
 "fmt"
 "sort"
 "strings"
 "sync"
)

// Connection identifies a selectable endpoint of an infrastructure provider.
// Target is provider-specific (a kubeconfig context for Kubernetes).
type Connection struct {
 Provider string `json:"provider"`
 ID string `json:"id"`
 Target string `json:"target"`
 Default bool `json:"default,omitempty"`
}

func (c Connection) QualifiedID() string { return c.Provider + "/" + c.ID }

type connectionKey struct{}

func WithConnection(ctx context.Context, providerID, connectionID string) context.Context {
 current, _ := ctx.Value(connectionKey{}).(map[string]string)
 next := make(map[string]string,len(current)+1)
 for k,v := range current {next[k]=v}
 next[providerID]=connectionID
 return context.WithValue(ctx,connectionKey{},next)
}
func ConnectionFromContext(ctx context.Context, providerID string) (string,bool) {
 selections,_:=ctx.Value(connectionKey{}).(map[string]string)
 value,ok:=selections[providerID]
 return value,ok && value!=""
}

type ConnectionDiscovery interface {
 Connections(context.Context)([]Connection,error)
}

func (r *Registry) Connections(ctx context.Context)([]Connection,error) {
 r.mu.RLock()
 providers:=make([]Provider,0,len(r.providers))
 for _,p:=range r.providers{providers=append(providers,p)}
 r.mu.RUnlock()
 result:=[]Connection{}
 for _,p:=range providers{
  discover,ok:=p.(ConnectionDiscovery)
  if !ok{continue}
  entries,err:=discover.Connections(ctx)
  if err!=nil{return nil,fmt.Errorf("discover %s connections: %w",p.ID(),err)}
  for _,c:=range entries{
   if c.Provider!=p.ID() || c.ID=="" || strings.Contains(c.ID,"/") || c.Target=="" {
    return nil,fmt.Errorf("invalid connection from provider %s",p.ID())
   }
   result=append(result,c)
  }
 }
 sort.Slice(result,func(i,j int)bool{return result[i].QualifiedID()<result[j].QualifiedID()})
 return result,nil
}

// Session is private to one operator. It never edits provider credentials/config.
type Session struct {
 mu sync.RWMutex
 selected map[string]string
}
func NewSession()*Session{return &Session{selected:map[string]string{}}}
func(s *Session) Clear(providerID string){
 s.mu.Lock();delete(s.selected,providerID);s.mu.Unlock()
}
func(s *Session) Selected() map[string]string{
 s.mu.RLock();defer s.mu.RUnlock()
 out:=map[string]string{}
 for key,value:=range s.selected {out[key]=value}
 return out
}
func(s *Session) Current(providerID string)string{
 s.mu.RLock();defer s.mu.RUnlock()
 return s.selected[providerID]
}
func(s *Session) Bind(ctx context.Context)context.Context{
 s.mu.RLock()
 defer s.mu.RUnlock()
 for p,c:=range s.selected{ctx=WithConnection(ctx,p,c)}
 return ctx
}
func(s *Session) Select(ctx context.Context,r *Registry,providerID,connectionID string)error{
 if providerID=="" || connectionID==""{return errors.New("provider and connection are required")}
 connections,err:=r.Connections(ctx);if err!=nil{return err}
 for _,c:=range connections {
  if c.Provider==providerID && c.ID==connectionID {
   s.mu.Lock()
   if s.selected==nil{s.selected=map[string]string{}}
   s.selected[providerID]=c.Target
   s.mu.Unlock()
   return nil
  }
 }
 return fmt.Errorf("unknown connection %s/%s",providerID,connectionID)
}
func(s *Session) SelectDefaults(ctx context.Context,r *Registry)error{
 connections,err:=r.Connections(ctx);if err!=nil{return err}
 for _,c:=range connections {
  if c.Default && s.Current(c.Provider)=="" {
   s.mu.Lock()
   if s.selected==nil{s.selected=map[string]string{}}
   s.selected[c.Provider]=c.Target
   s.mu.Unlock()
  }
 }
 return nil
}
