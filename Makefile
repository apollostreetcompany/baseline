.PHONY: build test web-typecheck web-dev cf-deploy-readback mac-build package-test plugin-validate release-build analytics-report verify verify-all

build:
	go build -o bin/baseline ./cmd/baseline

test:
	go test ./...

web-typecheck:
	cd web && npm run typecheck

web-dev:
	cd web && npm run dev

cf-deploy-readback:
	cf auth whoami
	cf workers deployments list --script-name baseline-ai

mac-build:
	cd macos/BaselineHotspots && swift build -Xswiftc -strict-concurrency=complete

package-test:
	cd package && npm test

plugin-validate:
	bash scripts/validate-codex-plugin.sh

release-build:
	bash scripts/build-release.sh

analytics-report:
	bash scripts/datafast-funnel-report.sh

verify: test web-typecheck package-test

verify-all: verify mac-build
