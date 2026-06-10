# Baseline Codex Plugin

This is the release-oriented Codex plugin bundle for Baseline. It contributes a
validated plugin manifest, the local Baseline MCP server declaration, and the
`baseline-health` skill for operator-safe health checks.

## Install Locally

Install the Baseline CLI first:

```sh
curl -fsSL https://trackbaseline.com/install.sh | sh
baseline --version
baseline doctor
```

`baseline --version` should print `baseline 0.1.0`. `baseline doctor` is a read-only preflight smoke; it does not send agent probes. Use `baseline setup` only when the operator is ready to write local Baseline state and start the first configured target eval.

Then add the repo-local marketplace from the repository root:

```sh
codex plugin marketplace add .agents/plugins
```

In the Codex app, install the `baseline` plugin from the `Baseline Plugin
Preview` marketplace.

## MCP Server

The plugin expects this command to be on `PATH`:

```sh
baseline serve mcp
```

If Codex cannot start the MCP server, first run `baseline --version` in the same environment. If the command is missing, install the CLI with the public install script or point Codex at a built binary; do not add extra MCP tools for version/preflight.

The MCP surface advertises exactly seven tools:

- `baseline_setup`
- `baseline_run`
- `baseline_doctor`
- `baseline_report`
- `baseline_accept`
- `baseline_schedule`
- `baseline_scrub_preview`

## Release Artifact

`scripts/build-release.sh` packages this directory as
`dist/baseline-codex-plugin.tgz`. The plugin tarball is a companion artifact to
the CLI binaries, not a replacement for the CLI binary itself.
