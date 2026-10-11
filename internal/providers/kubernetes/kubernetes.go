package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"example.com/support-shell/internal/provider"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// Kubernetes uses explicit per-session kubeconfig contexts; WASM never receives credentials.
func New() *provider.Catalog {
	r := provider.NewCatalog("kubernetes")
	r.SetConnectionDiscovery(discoverConnections)
	entries := []struct {
		id, path, resource string
		args               []string
	}{
		{"provider.kubernetes.contexts", "k8s contexts", "config get-contexts", nil},
		{"provider.kubernetes.nodes", "k8s nodes", "get nodes -o json", nil},
		{"provider.kubernetes.namespaces", "k8s namespaces", "get namespaces -o json", nil},
		{"provider.kubernetes.pods", "k8s pods", "get pods -o json", []string{"namespace"}},
		{"provider.kubernetes.events", "k8s events", "get events -o json", []string{"namespace"}},
		{"provider.kubernetes.pod.describe", "k8s pod describe", "describe pod", []string{"namespace", "pod"}},
		{"provider.kubernetes.pod.logs", "k8s pod logs", "logs", []string{"namespace", "pod"}},
	}
	registerPodDiagnostics(r)
	for _, entry := range entries {
		e := entry
		cmd := provider.Operation{ID: e.id, Path: e.path, Description: "Read-only Kubernetes: " + e.path, Risk: "read", Args: e.args}
		if e.id == "provider.kubernetes.pod.logs" || e.id == "provider.kubernetes.pod.describe" {
			cmd.Completer = completePodCommand(e.path)
		} else if e.id == "provider.kubernetes.pods" || e.id == "provider.kubernetes.events" {
			cmd.Completer = completeNamespaces(e.path)
		}
		cmd.Handler = func(ctx context.Context, in map[string]any) (any, error) {
			if e.id == "provider.kubernetes.contexts" {
				return contextTable(ctx)
			}
			if e.id == "provider.kubernetes.pods" || e.id == "provider.kubernetes.events" {
				ns, _ := in["namespace"].(string)
				if strings.HasPrefix(ns, "-") {
					return nil, fmt.Errorf("invalid namespace")
				}
				if ns == "" {
					ns = "default"
				}
				return kubectlJSON(ctx, "get", strings.TrimPrefix(e.id, "provider.kubernetes."), "-n", ns, "-o", "json")
			}
			t, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			argv := strings.Fields(e.resource)
			if len(e.args) > 0 {
				ns, _ := in["namespace"].(string)
				if ns == "" {
					return nil, fmt.Errorf("namespace required")
				}
				if strings.HasPrefix(ns, "-") {
					return nil, fmt.Errorf("invalid namespace")
				}
				switch e.id {
				case "provider.kubernetes.pods", "provider.kubernetes.events":
					argv = append(argv, "-n", ns)
				default:
					pod, _ := in["pod"].(string)
					if pod == "" || strings.HasPrefix(pod, "-") {
						return nil, fmt.Errorf("valid pod required")
					}
					argv = append(argv, pod, "-n", ns)
					if e.id == "provider.kubernetes.pod.logs" {
						for _, flag := range []string{"tail", "since", "container"} {
							if v, ok := in[flag].(string); ok && v != "" {
								if strings.HasPrefix(v, "-") {
									return nil, fmt.Errorf("invalid --%s", flag)
								}
								argv = append(argv, "--"+flag+"="+v)
							}
						}
						for _, flag := range []string{"previous", "timestamps"} {
							if v, ok := in[flag].(string); ok && (v == "true" || v == "1") {
								argv = append(argv, "--"+flag+"=true")
							}
						}
						if _, ok := in["tail"]; !ok {
							argv = append(argv, "--tail=150")
						}
					}
				}
			}
			argv, err := withKubeContext(ctx, argv)
			if err != nil { return nil, err }
			cmd := exec.CommandContext(t, "kubectl", argv...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err != nil {
				detail := strings.TrimSpace(stderr.String())
				if detail == "" {
					detail = strings.TrimSpace(stdout.String())
				}
				if detail == "" {
					detail = err.Error()
				}
				return nil, fmt.Errorf("kubectl %s failed: %s", strings.Join(argv, " "), detail)
			}
			data := stdout.Bytes()
			if len(data) > 1024*1024 {
				return nil, fmt.Errorf("output too large")
			}
			var val any
			if json.Unmarshal(data, &val) == nil {
				return val, nil
			}
			return strings.TrimSpace(string(data)), nil
		}
		_ = r.Add(cmd)
	}
	return r
}

// Context listing never returns auth material from kubeconfig.
type contextView struct {
	Name    string `json:"name"`
	Context struct {
		Cluster   string `json:"cluster"`
		User      string `json:"user"`
		Namespace string `json:"namespace"`
	} `json:"context"`
}
type kubeView struct {
	Current  string        `json:"current-context"`
	Contexts []contextView `json:"contexts"`
}

func withKubeContext(ctx context.Context, argv []string)([]string,error){
 if len(argv)>0 && argv[0]=="config" {return argv,nil}
 selected,ok:=provider.ConnectionFromContext(ctx,"kubernetes")
 if !ok {return nil,fmt.Errorf("no active Kubernetes connection; choose one with connections use kubernetes/<name>")}
 return append([]string{"--context",selected},argv...),nil
}
func runKubectl(ctx context.Context, args ...string) ([]byte, error) {
 t, cancel := context.WithTimeout(ctx, 10*time.Second)
 defer cancel()
 argv,err:=withKubeContext(ctx,args)
 if err!=nil{return nil,err}
 cmd := exec.CommandContext(t,"kubectl",argv...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl %s: %s", strings.Join(argv, " "), strings.TrimSpace(errBuf.String()))
	}
	if len(b) > 1024*1024 {
		return nil, fmt.Errorf("kubectl output too large")
	}
	return b, nil
}
func loadContexts(ctx context.Context) (kubeView, error) {
	b, err := runKubectl(ctx, "config", "view", "-o=json")
	if err != nil {
		return kubeView{}, err
	}
	var v kubeView
	if err = json.Unmarshal(b, &v); err != nil {
		return kubeView{}, fmt.Errorf("parse kubeconfig: %w", err)
	}
	sort.Slice(v.Contexts, func(i, j int) bool { return v.Contexts[i].Name < v.Contexts[j].Name })
	return v, nil
}
func contextTable(ctx context.Context) (string, error) {
	v, err := loadContexts(ctx)
	if err != nil {
		return "", err
	}
	if len(v.Contexts) == 0 {
		return "No Kubernetes contexts found", nil
	}
	if selected,ok:=provider.ConnectionFromContext(ctx,"kubernetes");ok{v.Current=selected}
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "CURRENT\tNAME\tCLUSTER\tNAMESPACE")
	for _, c := range v.Contexts {
		mark := " "
		if c.Name == v.Current {
			mark = "*"
		}
		ns := c.Context.Namespace
		if ns == "" {
			ns = "default"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", mark, c.Name, c.Context.Cluster, ns)
	}
	_ = w.Flush()
	return strings.TrimSuffix(buf.String(), "\n"), nil
}
func discoverConnections(ctx context.Context)([]provider.Connection,error){
 v,err:=loadContexts(ctx)
 if err!=nil{return nil,err}
 connections:=make([]provider.Connection,0,len(v.Contexts))
 for _,c:=range v.Contexts{
  connections=append(connections,provider.Connection{Provider:"kubernetes",ID:c.Name,Target:c.Name,Default:c.Name==v.Current})
 }
 return connections,nil
}

// Completion uses bounded, read-only kubectl calls; errors simply yield no suggestions.
func kubectlJSON(ctx context.Context, args ...string) (any, error) {
	b, err := runKubectl(ctx, args...)
	if err != nil {
		return nil, err
	}
	var result any
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func resourceNames(ctx context.Context, resource, ns string) []string {
	args := []string{"get", resource, "-o", "json"}
	if ns != "" {
		args = append(args, "-n", ns)
	}
	data, err := kubectlJSON(ctx, args...)
	if err != nil {
		return nil
	}
	obj, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	arr, _ := obj["items"].([]any)
	var out []string
	for _, item := range arr {
		v, ok := item.(map[string]any)
		if !ok {
			continue
		}
		meta, _ := v["metadata"].(map[string]any)
		if name, ok := meta["name"].(string); ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
func completeNamespaces(path string) func(context.Context, string) []string {
	return func(ctx context.Context, input string) []string {
		if strings.Contains(strings.TrimSpace(input), " ") {
			return nil
		}
		var out []string
		for _, name := range resourceNames(ctx, "namespaces", "") {
			if strings.HasPrefix(name, input) {
				out = append(out, path+" "+name)
			}
		}
		return out
	}
}
func completePodCommand(path string) func(context.Context, string) []string {
	return func(ctx context.Context, input string) []string {
		fields := strings.Fields(input)
		if len(fields) == 0 || (len(fields) == 1 && !strings.HasSuffix(input, " ")) {
			prefix := strings.TrimSpace(input)
			var out []string
			for _, ns := range resourceNames(ctx, "namespaces", "") {
				if strings.HasPrefix(ns, prefix) {
					out = append(out, path+" "+ns)
				}
			}
			return out
		}
		ns := fields[0]
		if len(fields) > 2 || (len(fields) == 2 && strings.HasSuffix(input, " ")) {
			if path == "k8s pod logs" && len(fields) >= 2 {
				var out []string
				pre := ""
				if len(fields) > 2 {
					pre = fields[len(fields)-1]
				}
				for _, flag := range []string{"--tail=", "--since=", "--container=", "--previous=true", "--timestamps=true"} {
					if strings.HasPrefix(flag, pre) {
						out = append(out, path+" "+ns+" "+fields[1]+" "+flag)
					}
				}
				return out
			}
			return nil
		}
		pre := ""
		if len(fields) == 2 {
			pre = fields[1]
		}
		var out []string
		for _, pod := range resourceNames(ctx, "pods", ns) {
			if strings.HasPrefix(pod, pre) {
				out = append(out, path+" "+ns+" "+pod)
			}
		}
		return out
	}
}
