package modules

import (
	"context"
	"example.com/support-shell/internal/core"
	"fmt"
	"strings"
)

// Pod restart data is captured from the live PodStatus. Kubernetes normally keeps
// only the most recently terminated container status, not every historical restart.
func registerPodDiagnostics(r *core.Registry) {
	_ = r.Add(core.Command{ID: "k8s.pod.restarts", Path: "k8s pod restarts", Description: "Inspect container restart counts, last termination and configured probes; restart count is cumulative", Risk: "read", Args: []string{"namespace", "pod"}, Completer: completePodCommand("k8s pod restarts"), Handler: func(ctx context.Context, in map[string]any) (any, error) {
		ns, pod, err := podScope(in)
		if err != nil {
			return nil, err
		}
		raw, err := kubectlJSON(ctx, "get", "pod", pod, "-n", ns, "-o", "json")
		if err != nil {
			return nil, err
		}
		root, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid pod response")
		}
		spec := obj(root, "spec")
		status := obj(root, "status")
		configs := map[string]map[string]any{}
		for _, kind := range []string{"containers", "initContainers"} {
			for _, v := range arr(spec, kind) {
				c, ok := v.(map[string]any)
				if !ok {
					continue
				}
				name, _ := c["name"].(string)
				configs[name] = c
			}
		}
		var rows []map[string]any
		for _, kind := range []string{"containerStatuses", "initContainerStatuses"} {
			for _, v := range arr(status, kind) {
				st, ok := v.(map[string]any)
				if !ok {
					continue
				}
				name, _ := st["name"].(string)
				last := obj(st, "lastState")
				term := obj(last, "terminated")
				cfg := configs[name]
				record := map[string]any{"container": name, "ready": st["ready"], "restartCount": st["restartCount"], "state": st["state"], "lastTermination": term, "livenessProbe": cfg["livenessProbe"], "startupProbe": cfg["startupProbe"], "readinessProbe": cfg["readinessProbe"]}
				rows = append(rows, record)
			}
		}
		return map[string]any{"source": "kubernetes PodStatus and PodSpec", "namespace": ns, "pod": pod, "node": spec["nodeName"], "phase": status["phase"], "containers": rows, "limitations": []string{"restartCount is cumulative and does not establish current restart frequency", "lastTermination describes only the most recent prior termination; it does not prove a liveness-probe failure", "probe configuration is not evidence of a probe-triggered restart"}}, nil
	}})
	_ = r.Add(core.Command{ID: "k8s.node.describe", Path: "k8s node describe", Description: "Inspect Kubernetes node conditions and events", Risk: "read", Args: []string{"node"}, Handler: func(ctx context.Context, in map[string]any) (any, error) {
		node, _ := in["node"].(string)
		if node == "" || strings.HasPrefix(node, "-") {
			return nil, fmt.Errorf("valid node required")
		}
		b, e := runKubectl(ctx, "describe", "node", node)
		return string(b), e
	}})
	_ = r.Add(core.Command{ID: "k8s.top.nodes", Path: "k8s top nodes", Description: "Read current node CPU and memory usage (metrics-server required)", Risk: "read", Handler: func(ctx context.Context, _ map[string]any) (any, error) {
		b, e := runKubectl(ctx, "top", "nodes")
		return string(b), e
	}})
	_ = r.Add(core.Command{ID: "k8s.top.pods", Path: "k8s top pods", Description: "Read current pod resource usage in a namespace", Risk: "read", Args: []string{"namespace"}, Completer: completeNamespaces("k8s top pods"), Handler: func(ctx context.Context, in map[string]any) (any, error) {
		ns, _ := in["namespace"].(string)
		if ns == "" || strings.HasPrefix(ns, "-") {
			return nil, fmt.Errorf("valid namespace required")
		}
		b, e := runKubectl(ctx, "top", "pods", "-n", ns)
		return string(b), e
	}})
}
func podScope(in map[string]any) (string, string, error) {
	ns, _ := in["namespace"].(string)
	pod, _ := in["pod"].(string)
	if ns == "" || pod == "" || strings.HasPrefix(ns, "-") || strings.HasPrefix(pod, "-") {
		return "", "", fmt.Errorf("namespace and pod required")
	}
	return ns, pod, nil
}
func obj(m map[string]any, k string) map[string]any { v, _ := m[k].(map[string]any); return v }
func arr(m map[string]any, k string) []any          { v, _ := m[k].([]any); return v }

// Used to accept both "true" and "false" as explicit flag values in the shell.
func boolFlag(v any) bool { return v == "true" || v == "1" || v == true }
