.PHONY: build test web-typecheck web-dev mac-build package-test release-build verify verify-all

build:
	go build -o bin/baseline ./cmd/baseline

test:
	go test ./...

web-typecheck:
	cd web && npm run typecheck

web-dev:
	cd web && npm run dev

mac-build:
	cd macos/BaselineHotspots && swift build -Xswiftc -strict-concurrency=complete

package-test:
	cd package && npm test

release-build:
	bash scripts/build-release.sh

verify: test web-typecheck package-test

verify-all: verify mac-build
