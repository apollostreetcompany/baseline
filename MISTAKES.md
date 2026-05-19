# MISTAKES.md - Baseline.ai

## Mistakes To Avoid
- Do not commit directly to `main`.
- Do not print payment, API, JWT, database, or private key secrets.
- Do not export raw agent prompts or outputs to cloud paths unless explicitly enabled.
- Do not let `baseline doctor` execute an agent; it remains read-only preflight.
- Do not treat static assets as available in Cloudflare Workers unless Wrangler asset configuration or another asset route is present.

## Session Lessons
- 2026-05-19: RepoPrompt workspace binding may be absent even when the repo exists locally. Bind or create the RepoPrompt workspace before code mapping.
- 2026-05-19: Wrangler uploaded an untracked `web/public/.DS_Store` asset during deploy. Add `.DS_Store` to `.gitignore`, remove stray macOS metadata before deploy, and confirm the Wrangler asset count.
- 2026-05-19: A redesign branch created from an older commit can roll back live Worker routes if deployed directly. Before deploy, compare against the currently deployed surface and preserve newer cloud/auth/MCP files from the active worktree.
- 2026-05-19: Local SwiftPM can pass while GitHub's newer Swift 6 toolchain rejects non-Sendable payloads. Run the macOS app build with `swift build -Xswiftc -strict-concurrency=complete`.
