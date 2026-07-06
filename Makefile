.PHONY: build/wasm

wasm_exec.js:
	cp "$(shell go env GOROOT)/lib/wasm/wasm_exec.js" ./internal/ui/static/

build/wasm: wasm_exec.js
	GOOS=js GOARCH=wasm go build -o ./internal/ui/static/bin/life.wasm ./cmd/life/life.go

run/web:
	go run ./cmd/web/
