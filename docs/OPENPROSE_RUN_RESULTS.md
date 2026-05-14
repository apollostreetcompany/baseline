# OpenProse VM Run Results

Ran the attached recipe-style `.prose.md` files after repairing the local Codex OpenProse skill to upstream 0.13.1. These files are legacy recipe contracts, not current `kind: service` / `kind: system` contracts, so they were executed through a compatibility activation with filesystem receipts.

## Runtime Repair

- Upstream source: https://github.com/openprose/prose/blob/main/skills/open-prose/SKILL.md
- Local stale backup: /Users/future/.codex/skills/open-prose.backup-20260513172352
- Repaired Codex skill: /Users/future/.codex/skills/open-prose
- Added missing current runtime docs: contract-markdown.md, forme.md, prosescript.md, responsibility-runtime.md, changelog.md

## Runs

| Run ID | Recipe | Output |
| --- | --- | --- |
| 20260514-002532-1-kill-gate | kill-gate | .prose/runs/20260514-002532-1-kill-gate/bindings/kill-gate/result.md |
| 20260514-002532-2-offer-smoke-test | offer-smoke-test | .prose/runs/20260514-002532-2-offer-smoke-test/bindings/offer-smoke-test/result.md |
| 20260514-002532-3-brand-to-landing | brand-to-landing | .prose/runs/20260514-002532-3-brand-to-landing/bindings/brand-to-landing/result.md |
| 20260514-002532-4-anti-slop-cleanup | anti-slop-cleanup | .prose/runs/20260514-002532-4-anti-slop-cleanup/bindings/anti-slop-cleanup/result.md |
| 20260514-002532-5-landing-page-responsive | landing-page-responsive | .prose/runs/20260514-002532-5-landing-page-responsive/bindings/landing-page-responsive/result.md |

## Consolidated Verdict

- Kill gate: GO for dogfooding, CONDITIONAL GO for paid SaaS until Stripe and 10-user pilot proof.
- Offer smoke test: PASS for dogfood, REFINE for paid launch because Stripe is not configured.
- Production landing: PASS for live dogfood; WARN for inactive checkout only.
- Anti-slop: PASS after screenshot-driven cleanup.
- Responsive: PASS at 390px, 768px, and desktop screenshots.

## Compatibility Note

Current OpenProse 0.13.1 expects runnable `.prose.md` sources to declare `kind: service` or `kind: system`. The attached skills-library recipes omit `kind`, so a strict `prose run` would report a structure error. The VM repair makes Codex current; the compatibility execution preserves these older recipes as runnable workflow receipts without mutating the upstream recipe files.
