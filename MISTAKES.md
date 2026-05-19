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
