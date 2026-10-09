use extism_pdk::*;
use serde_json::{json, Value};
use support_plugin_sdk::{Command, Manifest, Request, host_request};

#[host_fn]
extern "ExtismHost" { fn host_call(request: String) -> String; }

fn read_tool(id: &str, input: Value) -> FnResult<Value> {
    let result = unsafe { host_call(host_request(id, input))? };
    Ok(serde_json::from_str(&result)?)
}

#[plugin_fn]
pub fn describe(_: String) -> FnResult<String> {
    Ok(serde_json::to_string(&Manifest::new(
        "network", "0.1.0", &["k8s.nodes"],
        vec![Command::read("network.k8s.nodes", "network nodes",
            "Summarize Kubernetes node addresses and readiness via read-only Host API", &[])]
    ))?)
}

#[plugin_fn]
pub fn execute(input: String) -> FnResult<String> {
    let req = Request::parse(&input)?;
    if req.command != "network.k8s.nodes" {
        return Ok(json!({"error":"unknown command"}).to_string());
    }
    let nodes = read_tool("k8s.nodes", json!({}))?;
    if let Some(error) = nodes.get("error") {
        return Ok(json!({"error":error}).to_string());
    }
    let items: Vec<Value> = nodes.get("items").and_then(Value::as_array)
        .map(|rows| rows.iter().map(|node| {
            let addresses = node.pointer("/status/addresses").cloned().unwrap_or(json!([]));
            let conditions = node.pointer("/status/conditions").cloned().unwrap_or(json!([]));
            json!({"name":node.pointer("/metadata/name"),"addresses": addresses,
                "conditions":conditions})
        }).collect()).unwrap_or_default();
    Ok(json!({"source":"k8s.nodes","count":items.len(),"nodes":items}).to_string())
}

#[plugin_fn]
pub fn complete(_: String) -> FnResult<String> { Ok("[]".into()) }
