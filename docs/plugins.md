# Extism / WASM plugins

Support Shell loads a WASI plugin when built with `-tags extism` and `SUPPORT_PLUGIN` points to a WASM file. Its `describe()` response is validated as `pluginapi.Manifest` (`support.shell/v1alpha1`).

The plugin exports `describe`, `execute` and `complete`. Command metadata is registered in the shared Registry. `execute` receives a command ID and JSON input.

## Capabilities

A plugin declares requested Host operation IDs in `capabilities`. Host calls are permitted only if **both** requested and present in the built-in read-only allowlist; unknown capabilities prevent loading. This is not yet a general permissions framework or fully hardened sandbox. Credentials live in trusted Go modules.

## Demo

```bash
rustup target add wasm32-wasip1
make plugin
make run-plugin
```

In Shell, run `diagnose k8s kube-system`. The Rust diagnostics module calls `k8s.pods` and `k8s.events` through `host_call` and returns a structured summary with common provenance, not independent evidence.

The new manifest requires `apiVersion`, `name`, `version`, `commands` and declared `capabilities`. A previously built old WASM file must be recompiled. `pkg/pluginapi/README.md` documents the schema.


## Discovery and multiple plugins (PoC)

With `-tags extism`, the loader defaults to `PLUGIN_PATH=./plugins`. If the directory exists, it loads:
- `plugins/*.wasm` (prebuilt/distributed plugins)
- `plugins/*/target/wasm32-wasip1/release/*.wasm` (local Rust builds)

Use `PLUGIN_PATH=/opt/support/plugins` to load a different directory, or `PLUGIN_PATH=/opt/plugins/diagnostics.wasm` for one plugin. A comma-separated list of directories/files is also supported. For migration, `SUPPORT_PLUGINS` and `SUPPORT_PLUGIN` remain fallbacks when `PLUGIN_PATH` is unset. Discovery order is sorted. With the default directory missing or empty, loading silently succeeds with no plugins.

Each WASM module has its own Extism instance and capability map, and its calls into that instance are serialized. All plugin commands join the same registry; duplicate command IDs/paths fail registration and duplicate plugin names are rejected. The host restricts WASM calls to both declared capabilities and a read-only allowlist. This is a **startup-only loader**: no hot reload, signature verification or dependency resolution yet.

```bash
make deps
make plugin
make run-plugin
# or: PLUGIN_PATH=/opt/support/plugins make run-plugin
```

`make plugin` currently builds only the diagnostics example. To add another Rust plugin, place its source under `plugins/<name>`, export `describe/execute/complete`, and compile for `wasm32-wasip1`. Its release WASM is picked up on the next launch. The `extism` Go build tag is necessary.

## Plugin Manager

The built-in read-only CLI commands `plugins list` and `plugins info <name>` work with or without the Extism build tag. They expose the current in-process inventory of **successfully loaded** plugins: name, version, API version, WASM path, status (`loaded`), commands, and declared capabilities. With ordinary `make run` they return an empty list, because external WASM loading requires `-tags extism`.

```bash
make plugin
make run-plugin
# In the shell:
plugins list
plugins info diagnostics
```

This is not a persistent package manager: it does not yet support install/uninstall, plugin health checks, failed-plugin history, signatures, or hot reload. Startup still fails on invalid WASM plugins or duplicate registrations; the inventory is only populated after successful registration.

## Rust SDK и второй плагин

Общие JSON-типы для Rust-плагинов находятся в `plugins/sdk` (`support-plugin-sdk`): `Manifest`, `Command`, `Request`, `host_request()` и константа `API_VERSION`. SDK **не** содержит Extism runtime: точками входа `describe`, `execute`, `complete` управляет `extism-pdk` каждого плагина. Плагины используют локальную Cargo-зависимость `support-plugin-sdk = { path = "../sdk" }`.

Есть два примера независимых плагинов: `diagnostics` (`diagnose k8s <namespace>`, capabilities `k8s.pods` и `k8s.events`) и `network` (`network nodes`, capability `k8s.nodes`). После `make plugin && make run-plugin` оба автоматически обнаруживаются в `PLUGIN_PATH=./plugins` и доступны через `plugins list`.

Новый плагин: создайте `plugins/<name>/Cargo.toml` с `crate-type = ["cdylib"]`, добавьте `extism-pdk`, `serde_json` и локальный SDK. В `describe` верните `Manifest::new`, в `execute` разбирайте `Request::parse`, для Host API формируйте запрос `host_request`. Важно объявить только необходимые read-only capabilities; сейчас разрешения проверяются также встроенным allowlist Go Host.

```bash
make plugin
make run-plugin
# plugins list
# plugins info diagnostics
# plugins info network
# network nodes
```

CI запускает Rust SDK tests и отдельно компилирует оба WASM. Полноценное end-to-end тестирование исполнения через Extism по-прежнему требует runtime-проверки.

## Диагностические findings

Плагин `network` теперь предоставляет `network health`, который вызывает `k8s.nodes` один раз и самостоятельно проверяет NodeStatus. Ответ содержит `coverage` (total/ready/not_ready/unknown_ready), `findings_count`, массив `findings`, `source` и `note`. Каждая finding содержит `node`, `severity`, `code`, `message`, `source` и `condition`.

Правила: `Ready=False` → `critical`, отсутствие/неизвестность Ready → `warning`, Pressure=True → `warning`, отсутствие сведений о Pressure → `info`. `Ready=True` и все Pressure=False дают ноль findings. Это моментальный снимок, а не RCA и не историческая проверка; имена `CalicoIsUp` не выводятся, если они не анализировались.

```text
network health
set format json
network health
```

Результат пригоден для агентов и автоматизации; исходное `network nodes` сохранено. Unit-тесты анализа находятся в `plugins/network/src/analysis.rs`.

## Документация каждого плагина

Каждый плагин хранит собственный `README.md` рядом с `Cargo.toml`, с назначением, командами, примерами, запрашиваемыми capabilities и ограничениями. Текущие примеры: [Diagnostics](../plugins/diagnostics/README.md) и [Network](../plugins/network/README.md). Описание плагина не заменяет контракт `describe()`: команды и capabilities определяются манифестом WASM.

## Self-describing plugin manifests

Optional, backwards-compatible manifest fields now include `title`, `description`, `author` on the plugin, and `inputSchema`, `outputSchema`, `examples` on each command. Existing manifests with `apiVersion: support.shell/v1alpha1` remain accepted; the wire API version has **not** changed. Rust SDK exposes the corresponding optional metadata on `Manifest` and `Command`.

The same metadata feeds `plugins list` (short purpose), `plugins info NAME` (detail), `help <command path>` (arguments, examples and schemas), and the built-in AI tool definitions. Prefer JSON Schema object definitions with `properties` and `required`; use `additionalProperties: false` for tight tool interfaces. The Go Host currently validates basic schema shape in the manifest, not full input/output values at execution time. MCP exposure is a **planned adapter**, not implemented by this change.

Try `plugins list`, `plugins info network`, `help network health`, or `help diagnose k8s` after rebuilding the Rust plugins. Per-plugin README files remain the place for extended how-to and safety notes.
