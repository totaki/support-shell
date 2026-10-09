package modules

import (
	"bytes"
	"context"
	"encoding/json"
	"example.com/support-shell/internal/core"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// Kubernetes integration uses kubectl's configured authentication; WASM plugins never invoke it directly.
func RegisterKubernetes(r *core.Registry) {
	entries := []struct {
		id, path, resource string
		args               []string
	}{
		{"k8s.contexts", "k8s contexts", "config get-contexts", nil},
		{"k8s.nodes", "k8s nodes", "get nodes -o json", nil},
		{"k8s.namespaces", "k8s namespaces", "get namespaces -o json", nil},
		{"k8s.pods", "k8s pods", "get pods -o json", []string{"namespace"}},
		{"k8s.events", "k8s events", "get events -o json", []string{"namespace"}},
		{"k8s.pod.describe", "k8s pod describe", "describe pod", []string{"namespace", "pod"}},
		{"k8s.pod.logs", "k8s pod logs", "logs", []string{"namespace", "pod"}},
	}
	registerContexts(r)
	registerPodDiagnostics(r)
	for _, entry := range entries {
		e := entry
		cmd := core.Command{ID: e.id, Path: e.path, Description: "Read-only Kubernetes: " + e.path, Risk: "read", Args: e.args}
		if e.id == "k8s.pod.logs" || e.id == "k8s.pod.describe" {
			cmd.Completer = completePodCommand(e.path)
		} else if e.id == "k8s.pods" || e.id == "k8s.events" {
			cmd.Completer = completeNamespaces(e.path)
		}
		cmd.Handler = func(ctx context.Context, in map[string]any) (any, error) {
			if e.id == "k8s.contexts" {
				return contextTable(ctx)
			}
			if e.id == "k8s.pods" || e.id == "k8s.events" {
				ns, _ := in["namespace"].(string)
				if strings.HasPrefix(ns, "-") {
					return nil, fmt.Errorf("invalid namespace")
				}
				if ns == "" {
					ns = "default"
				}
				return kubectlJSON(ctx, "get", strings.TrimPrefix(e.id, "k8s."), "-n", ns, "-o", "json")
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
				case "k8s.pods", "k8s.events":
					argv = append(argv, "-n", ns)
				default:
					pod, _ := in["pod"].(string)
					if pod == "" || strings.HasPrefix(pod, "-") {
						return nil, fmt.Errorf("valid pod required")
					}
					argv = append(argv, pod, "-n", ns)
					if e.id == "k8s.pod.logs" {
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

func runKubectl(ctx context.Context, args ...string) ([]byte, error) {
	t, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(t, "kubectl", args...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl %s: %s", strings.Join(args, " "), strings.TrimSpace(errBuf.String()))
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
func registerContexts(r *core.Registry) {
	_ = r.Add(core.Command{ID: "k8s.context.use", Path: "k8s context use", Description: "Switch active kubeconfig context (confirmation required)", Risk: "mutate", Args: []string{"name"},
		Completer: func(ctx context.Context, prefix string) []string {
			v, err := loadContexts(ctx)
			if err != nil {
				return nil
			}
			var matches []string
			for _, c := range v.Contexts {
				if strings.HasPrefix(c.Name, prefix) {
					matches = append(matches, "k8s context use "+c.Name)
				}
			}
			return matches
		},
		Handler: func(ctx context.Context, in map[string]any) (any, error) {
			target, _ := in["name"].(string)
			if target == "" {
				return nil, fmt.Errorf("context name required")
			}
			v, err := loadContexts(ctx)
			if err != nil {
				return nil, err
			}
			found := false
			for _, c := range v.Contexts {
				if c.Name == target {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown context %q", target)
			}
			if target == v.Current {
				return "Already using context: " + target, nil
			}
			_, err = runKubectl(ctx, "config", "use-context", target)
			if err != nil {
				return nil, err
			}
			return "Switched context: " + v.Current + " → " + target, nil
		},
	})
	// Alias to keep singular and plural discoverable.
	_ = r.Add(core.Command{ID: "k8s.contexts.alias", Path: "k8s context list", Description: "List contexts in a table", Risk: "read", Handler: func(ctx context.Context, _ map[string]any) (any, error) { return contextTable(ctx) }})
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
