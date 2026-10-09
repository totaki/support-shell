use serde_json::{json, Value};

fn condition<'a>(node: &'a Value, kind: &str) -> Option<&'a Value> {
    node.pointer("/status/conditions")?.as_array()?.iter()
        .find(|c| c.get("type").and_then(Value::as_str) == Some(kind))
}

pub fn analyze(nodes: &Value) -> Result<Value, String> {
    if let Some(err) = nodes.get("error") { return Err(format!("k8s.nodes: {err}")); }
    let items = nodes.get("items").and_then(Value::as_array)
        .ok_or("k8s.nodes response has no items array")?;
    let mut findings: Vec<Value> = Vec::new();
    let mut ready = 0usize;
    let mut unknown = 0usize;
    let mut unhealthy = 0usize;
    for node in items {
        let name = node.pointer("/metadata/name").and_then(Value::as_str).unwrap_or("unknown");
        match condition(node, "Ready").and_then(|c| c.get("status")).and_then(Value::as_str) {
            Some("True") => ready += 1,
            Some("False") => {
                unhealthy += 1;
                findings.push(json!({"node":name,"severity":"critical","code":"NODE_NOT_READY",
                    "message":"Node Ready condition is False","source":"k8s.nodes","condition":"Ready"}));
            }
            _ => {
                unknown += 1;
                findings.push(json!({"node":name,"severity":"warning","code":"NODE_READY_UNKNOWN",
                    "message":"Node Ready condition is missing or Unknown","source":"k8s.nodes","condition":"Ready"}));
            }
        }
        for kind in ["MemoryPressure", "DiskPressure", "PIDPressure"] {
            match condition(node, kind).and_then(|c| c.get("status")).and_then(Value::as_str) {
                Some("True") => findings.push(json!({"node":name,"severity":"warning",
                    "code":format!("{}_ACTIVE",kind.to_uppercase()),
                    "message":format!("{kind} is True"),"source":"k8s.nodes","condition":kind})),
                Some("False") => {},
                _ => findings.push(json!({"node":name,"severity":"info",
                    "code":format!("{}_UNKNOWN",kind.to_uppercase()),
                    "message":format!("{kind} was not confirmed False"),"source":"k8s.nodes","condition":kind}))
            }
        }
    }
    Ok(json!({
        "source":"k8s.nodes","coverage":{"total":items.len(),"ready":ready,"not_ready":unhealthy,"unknown_ready":unknown},
        "findings":findings,"findings_count":findings.len(),
        "note":"Snapshot of Kubernetes NodeStatus; does not establish historical stability or root cause"
    }))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn healthy_nodes_have_no_findings() {
        let data=json!({"items":[{"metadata":{"name":"node-a"},"status":{"conditions":[
            {"type":"Ready","status":"True"},{"type":"MemoryPressure","status":"False"},
            {"type":"DiskPressure","status":"False"},{"type":"PIDPressure","status":"False"}]}}]});
        let output=analyze(&data).unwrap();
        assert_eq!(output["coverage"]["ready"],1);
        assert_eq!(output["findings_count"],0);
    }
    #[test]
    fn reports_not_ready_and_disk_pressure() {
        let data=json!({"items":[{"metadata":{"name":"node-b"},"status":{"conditions":[
            {"type":"Ready","status":"False"},{"type":"MemoryPressure","status":"False"},
            {"type":"DiskPressure","status":"True"},{"type":"PIDPressure","status":"False"}]}}]});
        let result=analyze(&data).unwrap();
        assert_eq!(result["findings_count"],2);
        assert_eq!(result["findings"][0]["code"],"NODE_NOT_READY");
        assert_eq!(result["findings"][1]["code"],"DISKPRESSURE_ACTIVE");
    }
    #[test]
    fn missing_conditions_are_unknown_not_healthy() {
        let data=json!({"items":[{"metadata":{"name":"node-c"},"status":{}}]});
        let result=analyze(&data).unwrap();
        assert_eq!(result["coverage"]["unknown_ready"],1);
        assert_eq!(result["findings_count"],4);
    }
    #[test]
    fn missing_items_is_error() { assert!(analyze(&json!({})).is_err()); }
}
