.PHONY: build/wasm css css/watch templ run/web

TAILWIND_VERSION := v4.3.2
TAILWIND := ./bin/tailwindcss
CSS_IN := ./internal/ui/styles/tailwind.css
CSS_OUT := ./internal/ui/static/css/app.css

wasm_exec.js:
	cp "$(shell go env GOROOT)/lib/wasm/wasm_exec.js" ./internal/ui/static/

build/wasm: wasm_exec.js
	GOOS=js GOARCH=wasm go build -o ./internal/ui/static/bin/life.wasm ./cmd/life/life.go

$(TAILWIND):
	mkdir -p bin
	curl -fsSL -o $@ https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-linux-x64
	chmod +x $@

# $(CSS_OUT) is committed: //go:embed needs it present for `go build` to work
# without the Tailwind binary.
css: $(TAILWIND)
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --minify

css/watch: $(TAILWIND)
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --watch

templ:
	go tool templ generate

run/web:
	go run ./cmd/web/
