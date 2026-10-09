use extism_pdk::*;
use serde::Deserialize;
use serde_json::{json, Value};

#[host_fn]
extern "ExtismHost" { fn host_call(request: String) -> String; }

#[derive(Deserialize)]
struct Request { command: String, input: Value }

fn call(id: &str, input: Value) -> FnResult<Value> {
    let req = json!({"command": id, "input": input});
    let raw = unsafe { host_call(req.to_string())? };
    let result: Value = serde_json::from_str(&raw)?;
    Ok(result)
}

#[plugin_fn]
pub fn describe(_: String) -> FnResult<String> {
    Ok(json!({"name":"diagnostics", "commands":[
        {"id":"diagnostics.k8s.inspect", "path":"diagnose k8s", "description":"Inspect live Kubernetes pods and events for a namespace via host read-only tools (same underlying source: kubectl)", "risk":"read", "args":["namespace"]}
    ]}).to_string())
}

#[plugin_fn]
pub fn execute(input: String) -> FnResult<String> {
    let req: Request = serde_json::from_str(&input)?;
    if req.command != "diagnostics.k8s.inspect" { return Ok(json!({"error":"unknown operation"}).to_string()); }
    let ns = req.input.get("namespace").and_then(Value::as_str).filter(|s| !s.is_empty()).unwrap_or("default");
    let pods = call("k8s.pods", json!({"namespace": ns}))?;
    let events = call("k8s.events", json!({"namespace": ns}))?;
    let problematic_pods: Vec<Value> = pods.get("items").and_then(Value::as_array).into_iter().flatten()
        .filter_map(|pod| {
            let name = pod.pointer("/metadata/name").and_then(Value::as_str).unwrap_or("unknown");
            let phase = pod.pointer("/status/phase").and_then(Value::as_str).unwrap_or("Unknown");
            let restarts: u64 = pod.pointer("/status/containerStatuses").and_then(Value::as_array)
                .map(|items| items.iter().map(|c| c.get("restartCount").and_then(Value::as_u64).unwrap_or(0)).sum()).unwrap_or(0);
            let waiting: Vec<&str> = pod.pointer("/status/containerStatuses").and_then(Value::as_array)
                .into_iter().flatten().filter_map(|c| c.pointer("/state/waiting/reason").and_then(Value::as_str)).collect();
            if phase != "Running" || restarts > 0 || !waiting.is_empty() {
                Some(json!({"name":name,"phase":phase,"restarts":restarts,"waiting":waiting}))
            } else { None }
        }).take(50).collect();
    let warnings: Vec<Value> = events.get("items").and_then(Value::as_array).into_iter().flatten()
        .filter(|e| e.get("type").and_then(Value::as_str) == Some("Warning"))
        .map(|e| json!({"reason":e.get("reason"),"message":e.get("message"),"object":e.pointer("/involvedObject/name")}))
        .take(30).collect();
    Ok(json!({"namespace":ns,
      "provenance":{"source":"kubectl", "operations":["k8s.pods", "k8s.events"], "independent_sources":false},
      "pod_count": pods.get("items").and_then(Value::as_array).map(|a|a.len()).unwrap_or(0),
      "problematic_pods":problematic_pods,"warning_events":warnings,
      "errors": ([pods.get("error"), events.get("error")].into_iter().flatten().collect::<Vec<_>>())
    }).to_string())
}

#[plugin_fn]
pub fn complete(_: String) -> FnResult<String> { Ok("[]".to_string()) }
