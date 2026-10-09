.PHONY: build run test plugin run-plugin
build:
	go build -o support ./cmd/support
run:
	go run ./cmd/support
test:
	go test ./...
plugin:
	cd plugins/diagnostics && cargo build --target wasm32-wasip1 --release
run-plugin:
	SUPPORT_PLUGIN=plugins/diagnostics/target/wasm32-wasip1/release/support_diagnostics.wasm go run -tags extism ./cmd/support
