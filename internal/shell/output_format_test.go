package shell

import (
 "strings"
 "testing"
)

func TestPluginFormats(t *testing.T){
 s:=&Shell{}
 value:=map[string]any{"plugins":[]map[string]any{{
  "name":"network","version":"0.1.0","status":"loaded","commands":[]any{map[string]any{"id":"network.nodes"}},
  "capabilities":[]any{"k8s.nodes"},
 }}}
 // JSON normalization mirrors the typed manager response.
 a:=s.renderResult("plugins.list",value)
 if !strings.Contains(a,"network")||!strings.Contains(a,"CAPABILITIES"){t.Fatalf("table: %s",a)}
 s.OutputFormat="json"
 b:=s.renderResult("plugins.list",value)
 if !strings.Contains(b,"\"plugins\"")||!strings.Contains(b,"\"network\""){t.Fatalf("json: %s",b)}
}

func TestPluginInfoReadable(t *testing.T){
 s:=&Shell{}
 detail:=map[string]any{"name":"network","version":"0.1.0","status":"loaded","path":"/plugins/network.wasm","apiVersion":"support.shell/v1alpha1","capabilities":[]any{"k8s.nodes"},"commands":[]any{map[string]any{"path":"network nodes","id":"network.k8s.nodes","risk":"read"}}}
 output:=s.renderResult("plugins.info",detail)
 if !strings.Contains(output,"Name: network")||!strings.Contains(output,"network nodes"){t.Fatalf("unexpected: %s",output)}
}

func TestNetworkHealthOutput(t *testing.T) {
    s := &Shell{}
    report := map[string]any{
        "coverage": map[string]any{"total": 3, "ready": 2, "not_ready": 1, "unknown_ready": 0},
        "findings": []any{map[string]any{"severity":"critical","node":"node-b","code":"NODE_NOT_READY","message":"Node Ready condition is False"}},
    }
    table := s.renderResult("network.k8s.health", report)
    if !strings.Contains(table, "SEVERITY") || !strings.Contains(table, "node-b") || strings.Contains(table, "\"coverage\"") {
        t.Fatalf("bad table output: %q", table)
    }
    s.OutputFormat = "json"
    raw := s.renderResult("network.k8s.health", report)
    if !strings.Contains(raw, "\"coverage\"") || !strings.Contains(raw, "\"findings\"") {
        t.Fatalf("bad JSON output: %q", raw)
    }
}
