package pluginapi

import "testing"

func TestManifestValidate(t *testing.T) {
    valid := Manifest{APIVersion: APIVersion, Name: "diagnostics", Version: "0.1.0", Commands: []Command{{ID: "diagnostics.k8s.inspect", Path: "diagnose k8s", Risk: RiskRead}}}
    if err := valid.Validate(); err != nil { t.Fatal(err) }
    cases := []struct{name string; change func(*Manifest)}{
        {"unsupported version", func(m *Manifest){m.APIVersion="support.shell/v999"}},
        {"missing name", func(m *Manifest){m.Name=""}},
        {"invalid risk", func(m *Manifest){m.Commands[0].Risk="unknown"}},
        {"duplicate id", func(m *Manifest){m.Commands=append(m.Commands, Command{ID:m.Commands[0].ID,Path:"other",Risk:RiskRead})}},
        {"duplicate capability", func(m *Manifest){m.Capabilities=[]string{"k8s.pods", "k8s.pods"}}},
        {"empty capability", func(m *Manifest){m.Capabilities=[]string{""}}},
        {"duplicate path", func(m *Manifest){m.Commands=append(m.Commands, Command{ID:"other",Path:m.Commands[0].Path,Risk:RiskRead})}},
    }
    for _, tc := range cases { t.Run(tc.name,func(t *testing.T){ m:=valid; m.Commands=append([]Command(nil),valid.Commands...);tc.change(&m);if err:=m.Validate();err==nil{t.Fatal("expected validation error")} }) }
}

func TestExtendedManifestMetadata(t *testing.T) {
 m := Manifest{APIVersion: APIVersion, Name:"network", Version:"0.2.0", Title:"Network", Description:"Inspect nodes", Commands:[]Command{{ID:"network.health",Path:"network health",Risk:RiskRead,InputSchema:map[string]any{"type":"object"},Examples:[]Example{{Command:"network health"}}}}}
 if err:=m.Validate();err!=nil {t.Fatal(err)}
 m.Commands[0].InputSchema=map[string]any{"type":"array"}
 if err:=m.Validate();err==nil {t.Fatal("schema type must be object")}
}
