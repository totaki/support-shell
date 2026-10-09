use extism_pdk::*;
use support_plugin_sdk::{Command, Manifest, Request, host_request};
use serde_json::{json, Value};

#[host_fn]
extern "ExtismHost" { fn host_call(request: String) -> String; }

fn call(id: &str, input: Value) -> FnResult<Value> {
    let raw = unsafe { host_call(host_request(id, input))? };
    let result: Value = serde_json::from_str(&raw)?;
    Ok(result)
}

#[plugin_fn]
pub fn describe(_: String) -> FnResult<String> {
    Ok(serde_json::to_string(&Manifest::new("diagnostics", "0.1.0",
        &["k8s.pods", "k8s.events"],
        vec![Command::read("diagnostics.k8s.inspect", "diagnose k8s",
            "Inspect Kubernetes pods and events", &["namespace"])]))?)
}

#[plugin_fn]
pub fn execute(input: String) -> FnResult<String> {
    let req = Request::parse(&input)?;
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
