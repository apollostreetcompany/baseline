# Publishing Baseline

Baseline has three distribution surfaces: the Go module/binary, the npm wrapper,
and the OpenClaw plugin bundle.

## Go Module

The module path must continue to match `go.mod`:

```sh
go mod tidy
go test ./...
git tag v0.1.0
git push origin v0.1.0
GOPROXY=proxy.golang.org go list -m github.com/apollostreetcompany/baseline@v0.1.0
```

Users can install the binary with:

```sh
go install github.com/apollostreetcompany/baseline/cmd/baseline@latest
```

## npm Wrapper

The package in `package/` publishes a `baseline` bin shim. It forwards to an
installed Go binary found through `BASELINE_BIN`, `./bin/baseline`, or `PATH`.

Pre-publish checks:

```sh
pnpm --dir package test
pnpm --dir package pack:dry-run
```

If `pnpm` is not installed locally, use Corepack or `npx pnpm@latest`:

```sh
npx --yes pnpm@latest --dir package test
npx --yes pnpm@latest --dir package pack --dry-run
```

Publish:

```sh
pnpm --dir package publish --access public
```

## OpenClaw Plugin

The bundle in `openclaw-plugin/` contains:

- `.codex-plugin/plugin.json`
- `.mcp.json`
- `skills/baseline-health/SKILL.md`

Local install and verification:

```sh
openclaw plugins install ./openclaw-plugin
openclaw plugins inspect baseline-ai
```

This bundle was smoke-tested with OpenClaw `2026.5.2`: `openclaw plugins inspect baseline-ai --json` reports bundle capabilities `skills` and `mcpServers`, with stdio MCP server `baseline`.

Restart the OpenClaw gateway after install. If plugin MCP import is unavailable,
register the MCP server directly:

```sh
openclaw mcp set baseline '{"command":"baseline","args":["serve","mcp"]}'
```

Tarball distribution:

```sh
tar -czf baseline-openclaw-plugin-v0.1.0.tgz -C openclaw-plugin .
openclaw plugins install ./baseline-openclaw-plugin-v0.1.0.tgz
```

## Smoke Tests

After installing any distribution, verify the same local command surface:

```sh
baseline init
baseline check --fast
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | baseline serve mcp
```

The MCP response should include `baseline_check`, `baseline_latest`,
`baseline_report`, `baseline_compare`, `baseline_mark_known_good`,
`baseline_config`, and `baseline_scrub_preview`.
