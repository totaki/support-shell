package shell

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"
)

func field(v any, names ...string) any {
	for _, n := range names {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[n]
	}
	return v
}
func str(v any) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprint(v)
}
func age(v any) string {
	t, err := time.Parse(time.RFC3339, str(v))
	if err != nil {
		return "-"
	}
	d := time.Since(t)
	if d < 0 {
		return "0s"
	}
	if d.Hours() >= 24 {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
func renderCommand(id string, value any) string {
	headers := []string{}
	rows := [][]string{}
	v, ok := value.(map[string]any)
	if !ok {
		return formatResult(value)
	}
	items, ok := v["items"].([]any)
	if !ok {
		return formatResult(value)
	}
	switch id {
	case "k8s.nodes":
		headers = []string{"NAME", "STATUS", "ROLES", "VERSION", "AGE"}
		for _, item := range items {
			labels, _ := field(item, "metadata", "labels").(map[string]any)
			roles := []string{}
			for k := range labels {
				if strings.HasPrefix(k, "node-role.kubernetes.io/") {
					role := strings.TrimPrefix(k, "node-role.kubernetes.io/")
					if role == "" {
						role = "control-plane"
					}
					roles = append(roles, role)
				}
			}
			if len(roles) == 0 {
				roles = []string{"<none>"}
			}
			status := "Unknown"
			if conditions, ok := field(item, "status", "conditions").([]any); ok {
				for _, c := range conditions {
					if field(c, "type") == "Ready" {
						if field(c, "status") == "True" {
							status = "Ready"
						} else {
							status = "NotReady"
						}
					}
				}
			}
			rows = append(rows, []string{str(field(item, "metadata", "name")), status, strings.Join(roles, ","), str(field(item, "status", "nodeInfo", "kubeletVersion")), age(field(item, "metadata", "creationTimestamp"))})
		}
	case "k8s.namespaces":
		headers = []string{"NAME", "STATUS", "AGE"}
		for _, item := range items {
			rows = append(rows, []string{str(field(item, "metadata", "name")), str(field(item, "status", "phase")), age(field(item, "metadata", "creationTimestamp"))})
		}
	case "k8s.pods":
		headers = []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
		for _, item := range items {
			statuses, _ := field(item, "status", "containerStatuses").([]any)
			ready := 0
			restarts := 0
			reason := str(field(item, "status", "phase"))
			for _, cs := range statuses {
				if field(cs, "ready") == true {
					ready++
				}
				if n, ok := field(cs, "restartCount").(float64); ok {
					restarts += int(n)
				}
				if r := field(cs, "state", "waiting", "reason"); r != nil {
					reason = str(r)
				}
			}
			rows = append(rows, []string{str(field(item, "metadata", "name")), fmt.Sprintf("%d/%d", ready, len(statuses)), reason, fmt.Sprint(restarts), age(field(item, "metadata", "creationTimestamp"))})
		}
	case "k8s.events":
		headers = []string{"TYPE", "REASON", "OBJECT", "MESSAGE"}
		for _, item := range items {
			rows = append(rows, []string{str(field(item, "type")), str(field(item, "reason")), str(field(item, "involvedObject", "name")), str(field(item, "message"))})
		}
	default:
		return formatResult(value)
	}
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
	if len(rows) == 0 {
		return "No resources found"
	}
	return strings.TrimSuffix(buf.String(), "\n")
}
