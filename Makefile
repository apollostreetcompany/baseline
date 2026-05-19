.PHONY: build test web-typecheck web-dev mac-build verify verify-all

build:
	go build -o bin/baseline ./cmd/baseline

test:
	go test ./...

web-typecheck:
	cd web && npm run typecheck

web-dev:
	cd web && npm run dev

mac-build:
	cd macos/BaselineHotspots && swift build

verify: test web-typecheck

verify-all: verify mac-build
