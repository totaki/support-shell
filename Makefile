PLUGIN_PATH ?= ./plugins

.PHONY: build run test deps plugin run-plugin test-extism wasm-target

build:
	go build -o support ./cmd/support

run:
	go run ./cmd/support

test:
	go test ./...

deps:
	go mod tidy

wasm-target:
	@rustup target list --installed | grep -qx wasm32-wasip1 || rustup target add wasm32-wasip1

plugin: wasm-target
	cargo build --manifest-path plugins/diagnostics/Cargo.toml --target wasm32-wasip1 --release
	cargo build --manifest-path plugins/network/Cargo.toml --target wasm32-wasip1 --release
	cargo build --manifest-path plugins/fixture/Cargo.toml --target wasm32-wasip1 --release

run-plugin:
	PLUGIN_PATH="$(PLUGIN_PATH)" go run -tags extism ./cmd/support

test-extism:
	go test -tags extism ./...

.PHONY: test-e2e

test-e2e: wasm-target
	cargo build --manifest-path plugins/fixture/Cargo.toml --target wasm32-wasip1 --release
	go build -tags extism -o support ./cmd/support
	python3 scripts/e2e_mcp.py
