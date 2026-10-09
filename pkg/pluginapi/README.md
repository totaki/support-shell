# Plugin API (v1alpha1)

This package defines the **versioned, JSON-serializable** manifest and request/response types for Support Shell plugins.

- `Manifest` declares plugin identity, commands and requested capabilities.
- `Command` describes CLI command paths, stable IDs and risk levels.
- `Request` / `Result` are transport-neutral envelopes.
- `Validate()` rejects incompatible API versions and malformed/duplicate commands.

**Migration status:** this is the first additive extraction of the SDK contract. The existing Extism adapter still has its legacy describe/execute protocol, and should be migrated in a separate change. This package does not yet implement capability enforcement; a declaration is not a permission grant.
