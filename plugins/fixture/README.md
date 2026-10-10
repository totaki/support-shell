# Fixture WASM plugin

The `fixture` plugin is a deterministic read-only test module for Support Shell. It does not access Kubernetes, the network or Host capabilities.

Its single command, `fixture echo <message>`, returns `{"echo":"<message>"}` for the provided string. The plugin declares a JSON input schema requiring `message` of type `string`; invalid input must be rejected by the Go Registry before WASM execution. This demonstrates that the same tool is callable through CLI and MCP.

Run `make test-e2e` from the repository root to compile the module, start the MCP stdio server and verify tool discovery, execution and invalid-argument rejection. You may also run `make plugin` and build the fixture separately for manual inspection.

This fixture is intended only for test environments, not as an operational diagnostic extension. It does not validate Kubernetes Host API permissions or external MCP-client integration.
