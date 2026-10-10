package core

// Surface identifies the interface invoking a tool.
type Surface string

const (
 SurfaceCLI Surface = "cli"
 SurfaceAgent Surface = "agent"
 SurfaceMCP Surface = "mcp"
)

type ToolPolicy struct {
 Deny map[Surface]map[string]bool
}

func (p ToolPolicy) Allowed(surface Surface, c Command) bool {
 if surface != SurfaceCLI && surface != SurfaceAgent && surface != SurfaceMCP {return false}
 if surface != SurfaceCLI && c.Risk != "read" {return false}
 if surface != SurfaceCLI && len(c.ID)>=13 && c.ID[:13]=="demo.cluster." {return false}
 return !p.Deny[surface][c.ID]
}
func (r *Registry) SetToolPolicy(p ToolPolicy) { r.policy=p }
func (r *Registry) Allowed(surface Surface, c Command) bool {return r.policy.Allowed(surface,c)}
func (r *Registry) ListFor(surface Surface) []Command {
 result:=[]Command{}
 for _,c:=range r.List(){if r.Allowed(surface,c){result=append(result,c)}}
 return result
}
