PLUGIN_PATH ?= ./plugins

.PHONY: build run test deps plugin run-plugin test-extism

build:
	go build -o support ./cmd/support

run:
	go run ./cmd/support

test:
	go test ./...

deps:
	go mod tidy

plugin:
	cargo build --manifest-path plugins/diagnostics/Cargo.toml --target wasm32-wasip1 --release
	cargo build --manifest-path plugins/network/Cargo.toml --target wasm32-wasip1 --release

run-plugin:
	PLUGIN_PATH="$(PLUGIN_PATH)" go run -tags extism ./cmd/support

test-extism:
	go test -tags extism ./...
