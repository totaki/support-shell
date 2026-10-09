//! JSON contract helpers shared by independently compiled WASM plugins.
//! Runtime exports remain provided by extism-pdk.
use serde::{Deserialize, Serialize};
use serde_json::Value;

pub const API_VERSION: &str = "support.shell/v1alpha1";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Command {
    pub id: String,
    pub path: String,
    pub description: String,
    pub risk: String,
    #[serde(default)]
    pub args: Vec<String>,
}
impl Command {
    pub fn read(id: &str, path: &str, description: &str, args: &[&str]) -> Self {
        Self { id: id.into(), path: path.into(), description: description.into(),
            risk: "read".into(), args: args.iter().map(|s| (*s).into()).collect() }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Manifest {
    pub api_version: String,
    pub name: String,
    pub version: String,
    pub capabilities: Vec<String>,
    pub commands: Vec<Command>,
}
impl Manifest {
    pub fn new(name: &str, version: &str, capabilities: &[&str], commands: Vec<Command>) -> Self {
        Self { api_version: API_VERSION.into(), name: name.into(), version: version.into(),
            capabilities: capabilities.iter().map(|s| (*s).into()).collect(), commands }
    }
}

#[derive(Debug, Deserialize)]
pub struct Request {
    pub command: String,
    #[serde(default)]
    pub input: Value,
}
impl Request {
    pub fn parse(input: &str) -> serde_json::Result<Self> { serde_json::from_str(input) }
}

pub fn host_request(command: &str, input: Value) -> String {
    serde_json::json!({"command": command, "input": input}).to_string()
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn manifest_and_request_wire_contract() {
        let m = Manifest::new("sample","0.1.0",&["k8s.nodes"],
            vec![Command::read("sample.nodes","sample nodes","List nodes",&[])]);
        let v = serde_json::to_value(m).unwrap();
        assert_eq!(v["apiVersion"], API_VERSION);
        assert_eq!(v["commands"][0]["risk"], "read");
        let req = Request::parse(r#"{"command":"sample.nodes","input":{}}"#).unwrap();
        assert_eq!(req.command, "sample.nodes");
        assert_eq!(serde_json::from_str::<Value>(&host_request("k8s.nodes", serde_json::json!({}))).unwrap()["command"],"k8s.nodes");
    }
}
