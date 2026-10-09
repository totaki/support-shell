package shell

import (
 "context"
 "sort"
 "strings"
)

// completions combines Shell built-ins and the shared command Registry.
// All returned candidates are complete input lines, not only suffixes.
func (s *Shell) completions(ctx context.Context, prefix string) []string {
 builtins:=[]string{"help","commands","history","report","report save","agent reset","set format table","set format json","exit","quit"}
 candidates:=append([]string{},builtins...)
 for _,cmd:=range s.Registry.List(){candidates=append(candidates,cmd.Path)}
 if strings.HasPrefix(prefix,"help ") {
  candidates=nil
  for _,cmd:=range s.Registry.List(){
   candidates=append(candidates,"help "+cmd.Path)
  }
 }
 if strings.HasPrefix(prefix,"set format ") {
  candidates=[]string{"set format table","set format json"}
 }
 if strings.HasPrefix(prefix,"plugins info ") {
  return uniqueCompletions(prefix,s.Registry.CompleteWithContext(ctx,prefix))
 }
 if !(strings.HasPrefix(prefix,"help ") || strings.HasPrefix(prefix,"set format ")) {
  candidates=append(candidates,s.Registry.CompleteWithContext(ctx,prefix)...)
 }
 return uniqueCompletions(prefix,candidates)
}
func uniqueCompletions(prefix string, values []string) []string {
 out:=[]string{}
 seen:=map[string]bool{}
 for _,s:=range values {
  if strings.HasPrefix(s,prefix) && !seen[s] {out=append(out,s);seen[s]=true}
 }
 sort.Strings(out)
 return out
}
