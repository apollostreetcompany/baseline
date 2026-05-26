# Publishing Baseline

Baseline has five distribution surfaces: the hosted install script, GitHub
Release binaries, the npm wrapper, the OpenClaw plugin bundle, and the Codex
plugin bundle. The binary is free to install; Pro billing gates cloud history,
workspace tokens, remote MCP account operations, monitoring, and retention.

## GitHub Release Binaries

The release workflow builds macOS and Linux tarballs, checksum files, the
OpenClaw plugin tarball, and the Codex plugin tarball whenever a `v*` tag is
pushed.

```sh
make verify-all
bash scripts/build-release.sh
git tag v0.1.0
git push origin v0.1.0
gh release view v0.1.0 --web
```

Artifacts:

- `baseline_Darwin_arm64.tar.gz`
- `baseline_Darwin_x86_64.tar.gz`
- `baseline_Linux_arm64.tar.gz`
- `baseline_Linux_x86_64.tar.gz`
- `baseline-openclaw-plugin.tgz`
- `baseline-codex-plugin.tgz`
- `checksums.txt`

The public install script at `https://trackbaseline.com/install.sh` downloads
from GitHub Releases, verifies the checksum entry, and installs to
`~/.local/bin` by default:

```sh
curl -fsSL https://trackbaseline.com/install.sh | sh
curl -fsSL https://trackbaseline.com/install.sh | BASELINE_INSTALL_DIR=/usr/local/bin sh
curl -fsSL https://trackbaseline.com/install.sh | BASELINE_VERSION=v0.1.0 sh
```

## npm Wrapper

The package in `package/` publishes a `baseline` bin shim. It forwards to an
installed binary found through `BASELINE_BIN`, `./bin/baseline`, or `PATH`. When
none is present, it downloads and verifies the matching GitHub Release asset,
then caches it under `~/.cache/baseline-ai/bin/<platform>-<arch>/baseline`.

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

Set `BASELINE_VERSION=v0.1.0` when testing the wrapper against a specific
release instead of `latest`.

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

Tarball distribution from the release artifact:

```sh
tar -czf baseline-openclaw-plugin-v0.1.0.tgz -C openclaw-plugin .
openclaw plugins install ./baseline-openclaw-plugin-v0.1.0.tgz
```

## Codex Plugin

The Codex plugin source lives in `plugins/baseline/` and contains:

- `.codex-plugin/plugin.json`
- `.mcp.json`
- `skills/baseline-health/SKILL.md`

Local validation:

```sh
make plugin-validate
```

Local development install:

```sh
codex plugin marketplace add .agents/plugins
```

The plugin requires the `baseline` CLI on `PATH`; install the CLI before first
MCP use. `scripts/build-release.sh` publishes the plugin source as
`dist/baseline-codex-plugin.tgz`.

## Smoke Tests

After installing any distribution, verify the same local command surface:

```sh
baseline setup
baseline doctor
baseline report
baseline accept <RUN_ID> --confirm "accept <RUN_ID>" --label clean-local
baseline compare
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | baseline serve mcp
```

The MCP response should include `baseline_setup`, `baseline_run`,
`baseline_doctor`, `baseline_report`, `baseline_accept`, `baseline_schedule`,
and `baseline_scrub_preview`.

For an OpenClaw runner smoke, use the normal eval path:

```sh
baseline run
baseline report
baseline accept <RUN_ID> --confirm "accept <RUN_ID>" --label clean-local
```

This path must use real OpenClaw message timing and token metadata when OpenClaw exposes it. `baseline run` defaults to the 14-question Baseline Core pack; wider packs require explicit operator approval through `--packs enabled` or `--packs all`. Missing session usage should be reported as unavailable, not estimated.
