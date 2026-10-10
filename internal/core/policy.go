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
