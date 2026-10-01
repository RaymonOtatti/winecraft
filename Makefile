# WineCraft (Go rebuild). Targets for later BUILD.md items fail until those items land.

ADDR    ?= 127.0.0.1:8787
WASM    := web/winecraft.wasm
GOROOT  := $(shell go env GOROOT)

.PHONY: test vet fuzz wasm size run client snap share clean

test:
	go test -race ./...

vet:
	go vet ./...

fuzz:
	go test -run=^$$ -fuzz=FuzzDecode -fuzztime=30s ./internal/proto

wasm:
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o $(WASM) ./cmd/client
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js

size: wasm
	@printf 'raw    %s bytes\n' "$$(wc -c < $(WASM) | tr -d ' ')"
	@printf 'gzip   %s bytes\n' "$$(gzip -9c $(WASM) | wc -c | tr -d ' ')"
	@command -v brotli >/dev/null && printf 'brotli %s bytes\n' "$$(brotli -c -q 11 $(WASM) | wc -c | tr -d ' ')" || true

run: wasm
	go run ./cmd/server -addr $(ADDR) -web web

client:
	go run ./cmd/client -server ws://$(ADDR)/ws

snap:
	mkdir -p snap
	go run ./cmd/client -snap snap/client.png

share:
	./scripts/share.sh

clean:
	rm -rf bin snap $(WASM) web/wasm_exec.js
