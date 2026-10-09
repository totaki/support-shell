// Package pluginapi defines the versioned, serializable contract for
// Support Shell plugins. It intentionally does not depend on Extism or Go handlers.
package pluginapi

import (
    "errors"
    "fmt"
    "strings"
)

const APIVersion = "support.shell/v1alpha1"

type Risk string
const (
    RiskRead Risk = "read"
    RiskWrite Risk = "write"
    RiskDangerous Risk = "dangerous"
)

// Manifest is returned by a plugin's describe() entrypoint.
type Manifest struct {
    APIVersion string `json:"apiVersion"`
    Name string `json:"name"`
    Version string `json:"version"`
    Commands []Command `json:"commands"`
    Capabilities []string `json:"capabilities,omitempty"`
}

type Command struct {
    ID string `json:"id"`
    Path string `json:"path"`
    Description string `json:"description"`
    Risk Risk `json:"risk"`
    Args []string `json:"args,omitempty"`
}

type Request struct {
    Command string `json:"command"`
    Input map[string]any `json:"input"`
}

type Result struct {
    Data any `json:"data,omitempty"`
    Error string `json:"error,omitempty"`
}

func (m Manifest) Validate() error {
    if m.APIVersion != APIVersion { return fmt.Errorf("unsupported plugin API version %q (want %q)", m.APIVersion, APIVersion) }
    if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Version) == "" { return errors.New("plugin name and version are required") }
    if len(m.Commands) == 0 { return errors.New("plugin must declare at least one command") }
    ids, paths := map[string]bool{}, map[string]bool{}
    for _, c := range m.Commands {
        if c.ID == "" || c.Path == "" { return errors.New("command id and path are required") }
        if c.Risk != RiskRead && c.Risk != RiskWrite && c.Risk != RiskDangerous { return fmt.Errorf("invalid risk %q for %s", c.Risk, c.ID) }
        if ids[c.ID] || paths[c.Path] { return fmt.Errorf("duplicate command id or path: %s", c.ID) }
        ids[c.ID], paths[c.Path] = true, true
    }
    capabilities := map[string]bool{}
    for _, capability := range m.Capabilities {
        if strings.TrimSpace(capability) == "" { return errors.New("empty capability") }
        if capabilities[capability] { return fmt.Errorf("duplicate capability %q", capability) }
        capabilities[capability] = true
    }
    return nil
}
