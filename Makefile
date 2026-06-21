.PHONY: build/wasm

wasm_exec.js:
	cp "$(shell go env GOROOT)/lib/wasm/wasm_exec.js" .

build/wasm: wasm_exec.js
	GOOS=js GOARCH=wasm go build -o ./bin/life.wasm ./cmd/wasm/life.go
