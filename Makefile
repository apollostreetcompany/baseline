.PHONY: build test web-typecheck web-dev verify

build:
	go build -o bin/baseline ./cmd/baseline

test:
	go test ./...

web-typecheck:
	cd web && npm run typecheck

web-dev:
	cd web && npm run dev

verify: test web-typecheck
