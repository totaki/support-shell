package shell

import (
 "context"
 "fmt"
 "os"
 "sort"
 "strings"

 "example.com/support-shell/internal/provider"
)

func (s *Shell) connectionContext(ctx context.Context) context.Context {
 if s.Providers==nil{return ctx}
 if s.Connections==nil{s.Connections=provider.NewSession()}
 if !s.connectionsInitialized {
  s.connectionsInitialized=true
  if err:=s.Connections.SelectDefaults(ctx,s.Providers);err!=nil{
   // The shell must remain usable when kubectl is unavailable.
   fmt.Fprintln(os.Stderr,"provider connection discovery:",err)
  }
  if name:=strings.TrimSpace(os.Getenv("SUPPORT_KUBERNETES_CONNECTION"));name!=""{
   if err:=s.Connections.Select(ctx,s.Providers,"kubernetes",name);err!=nil{
    s.Connections.Clear("kubernetes")
    fmt.Fprintln(os.Stderr,"provider connection:",err)
   }
  }
  if name:=strings.TrimSpace(os.Getenv("SUPPORT_OPENSTACK_CONNECTION"));name!=""{
   if err:=s.Connections.Select(ctx,s.Providers,"openstack",name);err!=nil{
    s.Connections.Clear("openstack")
    fmt.Fprintln(os.Stderr,"provider connection:",err)
   }
  }
 }
 s.syncConnectionLabel()
 return s.Connections.Bind(ctx)
}

func (s *Shell) syncConnectionLabel(){
 s.CurrentContext=""
 if s.Connections==nil{return}
 values:=s.Connections.Selected()
 names:=make([]string,0,len(values))
 for providerID,target:=range values{
  names=append(names,providerID+"/"+target)
 }
 sort.Strings(names)
 s.CurrentContext=strings.Join(names,",")
}

// printActiveConnections reports only the current selections; it does not
// discover/execute anything or expose cloud credentials.
func (s *Shell) printActiveConnections() {
 if s.Connections==nil {
  fmt.Println("No active connections")
  return
 }
 selected:=s.Connections.Selected()
 if len(selected)==0 {
  fmt.Println("No active connections")
  return
 }
 providers:=make([]string,0,len(selected))
 for name:=range selected{providers=append(providers,name)}
 sort.Strings(providers)
 for _,name:=range providers {
  fmt.Printf("%s: %s\n",name,selected[name])
 }
}

func (s *Shell) listConnections(ctx context.Context){
 if s.Providers==nil{fmt.Println("No provider registry");return}
 fmt.Println("CURRENT  CONNECTION")
 for _,providerID:=range s.Providers.IDs(){
  entries,err:=s.Providers.ConnectionsFor(ctx,providerID)
  if err!=nil{fmt.Fprintln(os.Stderr,"connections:",err);continue}
  for _,connection:=range entries{
   marker:=" "
   if s.Connections!=nil && s.Connections.Current(connection.Provider)==connection.Target{marker="*"}
   fmt.Printf("  %s      %s\n",marker,connection.QualifiedID())
  }
 }
}

func (s *Shell) queueConnection(ctx context.Context,raw string){
 if s.Providers==nil{fmt.Fprintln(os.Stderr,"connections: no provider registry");return}
 providerID,id,ok:=strings.Cut(raw,"/")
 if !ok||providerID==""||id==""{fmt.Fprintln(os.Stderr,"usage: connections use <provider>/<connection>");return}
 connections,err:=s.Providers.ConnectionsFor(ctx,providerID)
 if err!=nil{fmt.Fprintln(os.Stderr,"connections:",err);return}
 for _,c:=range connections{
  if c.Provider==providerID && c.ID==id{
   fmt.Printf("Select connection:\n  Current: %s\n  Target:  %s\n",s.CurrentContext,c.QualifiedID())
   s.PendingContext=c.QualifiedID()
   return
  }
 }
 fmt.Fprintln(os.Stderr,"unknown connection:",raw)
}
