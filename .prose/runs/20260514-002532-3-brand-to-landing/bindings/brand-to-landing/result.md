# Brand-to-Landing Output: Baseline.ai

1. Production URL: https://baseline-ai.ryan-borker.workers.dev

2. Brand-to-copy mapping:
- Promise: know when your agent got worse before it costs work -> hero copy.
- Audience: OpenClaw/Hermes/Codex/Claude Code users, agencies, CTOs -> explicit hero audience.
- Proof/pain: 5s vs 60s, 60% to 91%, 10-15 timed questions -> metric strip.
- Trust: local-first, no raw prompt export -> safety copy and docs.

3. Responsive audit: 390px PASS after screenshot; 768px PASS after tablet screenshot; 1440px PASS after layout correction.

4. Anti-slop audit: PASS. Removed generic feature grid energy; kept concrete product dashboard visual and exact pain language.

5. SEO/AEO audit: PASS. Metadata, robots, sitemap, and SoftwareApplication JSON-LD exist.

6. Legal checklist: Privacy and Terms routes exist and are linked. Cookie consent not added because no cookie-based analytics was implemented.

7. Analytics verification: /api/events exists and CTA beacons are wired; Neon baseline_events table exists.

8. Audit scorecard: Responsive PASS, Conversion WARN until Stripe configured, Copy/Simplicity PASS, Slop PASS, Legal PASS, SEO/AEO PASS.

9. Fix list: configure Stripe and add retention cleanup for 14-day promise.
