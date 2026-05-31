# Baseline Codex Plugin

## Readiness Assessment

Baseline was not production-ready as a Codex plugin before Bead 32. The product
already had the right runtime ingredients: a local CLI, a stdio MCP server,
agent-safe async run lifecycle, local report artifacts, and an OpenClaw plugin
bundle. The existing `openclaw-plugin/` bundle was close, but it did not satisfy
the current Codex plugin ingestion contract.

Current gaps in the legacy bundle:

- `openclaw-plugin/.codex-plugin/plugin.json` uses legacy fields (`mcp` and
  `publisher`) that the plugin validator rejects.
- The manifest `name` is display-style text instead of a normalized plugin
  identifier matching the plugin folder.
- The manifest lacks required `author` and `interface` metadata.
- The bundled skill lacks required YAML frontmatter.
- Release packaging only emitted `baseline-openclaw-plugin.tgz`, not a Codex
  plugin artifact.

## Version 1 Built In This Bead

The v1 Codex plugin now lives at `plugins/baseline/` and includes:

- `.codex-plugin/plugin.json` with validated plugin metadata.
- `.mcp.json` pointing Codex at `baseline serve mcp`.
- `skills/baseline-health/SKILL.md` with required frontmatter and operator-safe
  workflow guidance.
- A repo-local marketplace entry at `.agents/plugins/marketplace.json`.
- Release packaging for `dist/baseline-codex-plugin.tgz`.
- `make plugin-validate` for local validation with the Codex plugin-creator
  validator.

This is a valid local/development Codex plugin, assuming the Baseline CLI is
already installed and on `PATH`.

## Current First-Run Contract

Before installing or smoking the plugin, install the CLI and verify it without
starting a run:

```sh
curl -fsSL https://trackbaseline.com/install.sh | sh
baseline --version
baseline doctor
```

`baseline --version` should print `baseline 0.1.0`. `baseline doctor` is
read-only preflight and must not send agent probes. The first command that
writes Baseline local state and starts the configured target eval is
`baseline setup`; later drift checks use `baseline run`.

The plugin MCP server command is still `baseline serve mcp`. A healthy
`tools/list` response advertises exactly seven tools: `baseline_setup`,
`baseline_run`, `baseline_doctor`, `baseline_report`, `baseline_accept`,
`baseline_schedule`, and `baseline_scrub_preview`. If the CLI is missing or
stale, recover by installing the CLI or pointing Codex at a built binary; do not
add an MCP version/preflight tool.

## Productionization Gaps

The v1 plugin is not yet a fully productionized public plugin because the plugin
does not install or update the Baseline CLI by itself. The current MVP expects
users to install the CLI first through `https://trackbaseline.com/install.sh`,
GitHub Releases, npm, or a later Homebrew tap.

Missing production pieces:

- Public plugin distribution channel and marketplace policy decision.
- A packaged install/update path that verifies the CLI binary before first MCP
  use.
- CI validation for the plugin manifest using an official or vendored schema,
  not only the local plugin-creator skill validator.
- Plugin smoke tests in a clean Codex environment with no preinstalled
  `baseline` binary.
- Product icons/screenshots sized for Codex plugin presentation.
- Version coordination between the CLI release, MCP tool contract, skill text,
  and plugin artifact.
- User-facing recovery for missing CLI, stale binary, and unsupported platform.
- Signed/checksummed plugin artifact publication alongside release binaries.
- A decision about whether the legacy `openclaw-plugin/` bundle should remain
  separate or become a compatibility alias of the Codex plugin.

## Productionized Next Steps

1. Add a plugin preflight command that reports missing CLI, unsupported
   platform, stale version, and install instructions without starting a run.
2. Add a bundled installer script or Codex-compatible setup flow that downloads
   the matching release binary, verifies `checksums.txt`, and refuses to run on
   checksum mismatch.
3. Add Codex plugin validation to CI with a repo-local validator or official
   schema package.
4. Build a clean-environment smoke test: install marketplace, install plugin,
   run `tools/list`, run `baseline_doctor`, and verify the missing-CLI and
   installed-CLI paths.
5. Generate production plugin assets: icon, logo, and screenshots under
   `plugins/baseline/assets/`, then add those paths to `plugin.json`.
6. Publish `baseline-codex-plugin.tgz` in GitHub Releases and document the
   install/update flow on `trackbaseline.com`.
7. Decide whether to migrate `openclaw-plugin/` to the same manifest shape or
   keep it as an OpenClaw compatibility bundle with a separate release artifact.
8. Run a pilot with one fresh Codex install and one existing OpenClaw user, then
   update the skill text from observed failures before broad release.
