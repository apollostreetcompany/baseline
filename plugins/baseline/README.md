# Baseline Codex Plugin

This is the release-oriented Codex plugin bundle for Baseline. It contributes a
validated plugin manifest, the local Baseline MCP server declaration, and the
`baseline-health` skill for operator-safe health checks.

## Install Locally

Install the Baseline CLI first:

```sh
curl -fsSL https://trackbaseline.com/install.sh | sh
baseline doctor
```

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

The MCP surface advertises:

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
