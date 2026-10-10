use extism_pdk::*;
use serde_json::json;
use support_plugin_sdk::{Command, Example, Manifest, Request};

#[plugin_fn]
pub fn describe(_: String) -> FnResult<String> {
    let mut manifest = Manifest::new("fixture", "0.1.0", &[], vec![
        Command::read("fixture.echo", "fixture echo", "Echo a message for integration tests", &["message"]),
        Command::read("fixture.invalid", "fixture invalid", "Produce invalid JSON for host recovery tests", &[]),
    ]);
    manifest.description = "Deterministic WASM smoke-test tool without Kubernetes access".into();
    manifest.title = "E2E Fixture".into();
    manifest.commands[0].input_schema = Some(json!({
        "type": "object",
        "properties": {"message": {"type": "string"}},
        "required": ["message"],
        "additionalProperties": false
    }));
    manifest.commands[0].examples = vec![Example {
        command: "fixture echo hello".into(),
        description: "Return the same message".into(),
    }];
    Ok(serde_json::to_string(&manifest)?)
}

#[plugin_fn]
pub fn execute(input: String) -> FnResult<String> {
    let req = Request::parse(&input)?;
    if req.command == "fixture.invalid" {
        return Ok("not-json".into());
    }
    if req.command != "fixture.echo" {
        return Ok(json!({"error":"unknown operation"}).to_string());
    }
    let message = req.input.get("message").and_then(|v|v.as_str()).unwrap_or("");
    Ok(json!({"echo":message}).to_string())
}

#[plugin_fn]
pub fn complete(_: String) -> FnResult<String> {
    Ok("[]".into())
}
