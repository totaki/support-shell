package shell

import (
 "strings"
 "example.com/support-shell/internal/core"
)

func commandHelp(c core.Command) string {
 lines:=[]string{c.Path, c.Description, "Risk: "+c.Risk}
 if len(c.Args)>0 { lines=append(lines,"Arguments: "+strings.Join(c.Args,", ")) }
 if len(c.Examples)>0 {lines=append(lines,"Examples:");for _,e:=range c.Examples {lines=append(lines,"  "+e)}}
 if c.InputSchema!=nil {lines=append(lines,"Input schema:",core.JSON(c.InputSchema))}
 if c.OutputSchema!=nil {lines=append(lines,"Output schema:",core.JSON(c.OutputSchema))}
 return strings.Join(lines,"\n")
}
