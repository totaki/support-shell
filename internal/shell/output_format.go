package shell

import (
 "bytes"
 "fmt"
 "sort"
 "strings"
 "text/tabwriter"
 "example.com/support-shell/internal/core"
)

func (s *Shell) renderResult(id string, value any) string {
 if s.OutputFormat == "json" { return core.JSON(value) }
 switch id {
 case "plugins.list":
  if data,ok:=value.(map[string]any);ok {
   if raw,ok:=data["plugins"];ok {
    rows:=[][]string{}
    switch plugins:=raw.(type) {
    case []map[string]any:
     for _,p:=range plugins {rows=append(rows, pluginRow(p))}
    default:
     // Plugin Manager returns a typed slice. JSON conversion keeps the view
     // independent of the manager package and avoids a package dependency cycle.
     decoded:=map[string]any{}
     if err:=decodeJSON(value,&decoded);err==nil {
      if records,ok:=decoded["plugins"].([]any);ok {for _,item:=range records {
       if p,ok:=item.(map[string]any);ok {rows=append(rows,pluginRow(p))}
      }}
     }
    }
    if len(rows)==0{return "No plugins loaded"}
    return tabulate([]string{"NAME","VERSION","STATUS","COMMANDS","CAPABILITIES"},rows)
   }
  }
 case "plugins.info":
  decoded:=map[string]any{}
  if err:=decodeJSON(value,&decoded);err==nil {
   header:=[]string{
    "Name: "+displayValue(decoded["name"]),
    "Version: "+displayValue(decoded["version"]),
    "API: "+displayValue(decoded["apiVersion"]),
    "Status: "+displayValue(decoded["status"]),
    "Path: "+displayValue(decoded["path"]),
   }
   if caps,ok:=decoded["capabilities"].([]any);ok{
    vals:=make([]string,0,len(caps));for _,c:=range caps{vals=append(vals,displayValue(c))}
    header=append(header,"Capabilities: "+strings.Join(vals,", "))
   }
   if cmds,ok:=decoded["commands"].([]any);ok&&len(cmds)>0{
    rows:=[][]string{}
    for _,c:=range cmds{if v,ok:=c.(map[string]any);ok{
     rows=append(rows,[]string{displayValue(v["path"]),displayValue(v["id"]),displayValue(v["risk"])})
    }}
    header=append(header,"Commands:",tabulate([]string{"PATH","ID","RISK"},rows))
   }
   return strings.Join(header,"\n")
  }
 }
 return renderCommand(id,value)
}
func pluginRow(p map[string]any) []string {
 count:="0"
 if c,ok:=p["commands"].([]any);ok{count=fmt.Sprint(len(c))}
 caps:=[]string{}
 if c,ok:=p["capabilities"].([]any);ok{for _,v:=range c{caps=append(caps,displayValue(v))}}
 sort.Strings(caps)
 return []string{displayValue(p["name"]),displayValue(p["version"]),displayValue(p["status"]),count,strings.Join(caps,", ")}
}
func displayValue(v any) string {if v==nil{return "-"};return fmt.Sprint(v)}
func tabulate(headings []string, rows [][]string) string {
 var buf bytes.Buffer
 w:=tabwriter.NewWriter(&buf,0,0,2,' ',0)
 fmt.Fprintln(w,strings.Join(headings,"\t"))
 for _,row:=range rows{fmt.Fprintln(w,strings.Join(row,"\t"))}
 _=w.Flush()
 return strings.TrimSpace(buf.String())
}
