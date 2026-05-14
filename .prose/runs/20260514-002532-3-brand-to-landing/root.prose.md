---
name: brand-to-landing
description: >-
  End-to-end recipe for turning a brand system into a live, audited, responsive landing page.
  Chains brand-os-studio into ui-workflow landing recipe, adds legal docs, deploys,
  instruments analytics, and runs multi-agent audit. Use when you need a production
  landing page from an existing or new brand. Do not use for product UI, dashboards,
  or logged-in app screens.
entry-stage: G0
target-fidelity: F5 (launch-ready)
use-when:
  - You have a brand (or are building one) and need a landing page
  - A smoke test passed and you need a production page
  - The existing landing page reads as AI-generated or doesn't convert
  - You want a full brand → deploy pipeline in one sitting
avoid-when:
  - The product is a logged-in app UI (use `recipes/22-ios-release-lane.prose.md` for iOS or a dedicated product UI workflow)
  - You only need copy edits on an existing page (use `skills/ui-clarify/SKILL.md` or another focused copy-edit skill directly)
  - You haven't validated the offer yet (run `recipes/03-offer-smoke-test.prose.md` first)
skills:
  - skills/brand-os-studio/SKILL.md (stages 1-4, or existing brand OS)
  - recipes/web-landing-page-responsive.prose.md
  - recipes/web-anti-slop-cleanup.prose.md
  - skills/frontend-design/SKILL.md
  - skills/viral-decoder/SKILL.md (Mode 3 for hook angles)
  - skills/content-architecture/SKILL.md (cluster planning if multi-page)
  - skills/legal-shield/SKILL.md
  - skills/deploy-ops/SKILL.md
  - skills/analytics-baseline/SKILL.md
  - skills/seo-aeo/SKILL.md (meta tags, schema markup)
  - skills/ai-seo/SKILL.md (AI extractability)
audit-agents:
  - Responsive Auditor
  - Conversion Auditor
  - Copy/Simplicity Auditor
  - Slop Auditor
  - Legal Compliance Auditor
  - SEO/AEO Auditor
inputs:
  required:
    - Brand OS or brand brief (if no OS exists, start with `skills/brand-os-studio/SKILL.md`)
    - Primary conversion goal (signup / purchase / waitlist / book call)
    - Target audience
  optional:
    - Existing landing page URL or screenshots
    - Competitor landing pages for reference
    - Price point
    - Analytics account (Datafast / GSC)
outputs:
  - Live landing page at production URL
  - Privacy policy + ToS hosted
  - Analytics instrumented
  - SEO meta + schema markup
  - Multi-agent audit scorecard
  - Specific fix list if audit fails
---

# Goal

Go from brand system to live, audited, converting landing page. One session, one pipeline.

# Workflow

## G0 — Brand Input

**If brand OS exists:**
Read the brand OS. Extract: value proposition, target user, core promise, proof points, voice/tone rules, anti-patterns, visual direction.

**If no brand OS exists:**
Run `skills/brand-os-studio/SKILL.md` in lightweight mode (stages 1-3 only):
1. Fascinate force-choice → lock personality
2. Brand idea territories → pick winner
3. Positioning → lock frame of reference + differentiation

Read `skills/brand-os-studio/SKILL.md` for full instructions. JiT load `skills/brand-os-studio/references/fascinate-force-choice.md` and `skills/brand-os-studio/references/positioning-research.md`.

**Gate:** One-sentence value proposition must exist. If you can't state it clearly, stop and run `workflows/tibo/growth-orchestrator.prose` first.

## G1 — Landing Copy Extraction

From the brand OS, extract landing page content blocks:

1. **Hero** — headline (6-12 words) + support line (1-2 sentences) + primary CTA
2. **Problem agitation** — the painful current state, in audience language
3. **Solution frame** — what changes, positioned as the brand promise (not feature list)
4. **Proof** — testimonials, data points, credentials, or social proof
5. **How it works** — 3 steps max
6. **Objection handling** — address the top 2-3 reasons someone wouldn't act
7. **Final CTA** — repeat the primary action with urgency or clarity reinforcement

Run `skills/viral-decoder/SKILL.md` Mode 3 on the hero headline — generate 5 emotion-led hook variants. Pick the sharpest one.

**Gate:** Hero headline must pass the "would I stop scrolling?" test. If not, iterate.

## G2 — Responsive Design

Run `recipes/web-landing-page-responsive.prose.md` with the extracted copy:

- Capture current state (or wireframe) at 390px, 768px, 1280px
- G0-G1: Set direction, name the signature trait to preserve
- G2: Structure — hero order on mobile: headline → proof → CTA → media
- G3: Platform fit — responsive checks at all 3 breakpoints
- G4: Copy compression — word budgets per section
- G5: Motion — only after structure passes
- G6: Hardening — tap targets, overflow, metadata

Read `recipes/web-landing-page-responsive.prose.md` for full gate specs.

**Gate:** Mobile hero must fit without broken wrapping or clipped CTA.

## G3 — Anti-Slop Pass

Run `recipes/web-anti-slop-cleanup.prose.md`:

- Does this page immediately read as AI-generated?
- Kill: generic stock imagery, "unlock your potential" language, gratuitous gradients, icon grids that add no information
- Keep: brand-specific voice, real proof, specific numbers, honest tone

Read `recipes/web-anti-slop-cleanup.prose.md` for full methodology.

**Gate:** Slop Auditor must pass. If it reads as AI-made, iterate before proceeding.

## G4 — SEO + AI Discoverability

1. Run `skills/seo-aeo/SKILL.md`: meta title, meta description, OG tags, JSON-LD schema markup
2. Run `skills/ai-seo/SKILL.md`: direct answer block, stable entity naming, extractable summary
3. If multi-page site: run `skills/content-architecture/SKILL.md` for cluster planning and internal linking

Read `skills/seo-aeo/SKILL.md` strategy selection table. JiT load `skills/seo-aeo/references/schema-markup.md`.
Read `skills/ai-seo/SKILL.md` for LLM citation optimization.

**Gate:** Page has complete meta tags, at least one JSON-LD block, and an AI-extractable summary.

## G5 — Legal Compliance

Run `skills/legal-shield/SKILL.md`:

1. Generate privacy policy from template. JiT load `skills/legal-shield/assets/privacy-policy-template.md`.
2. Generate Terms of Service. JiT load `skills/legal-shield/assets/tos-template.md`.
3. Add cookie consent banner (if EU traffic expected)
4. Verify GDPR/CCPA checklist items

Read `skills/legal-shield/SKILL.md` for stack-specific customization (Stripe, Klaviyo, Datafast, Cloudflare).

**Gate:** Privacy policy and ToS are hosted and linked from page footer.

## G6 — Deploy + Analytics

1. Run `skills/deploy-ops/SKILL.md`: Cloudflare DNS + Vercel/Render hosting, SSL, custom domain
2. Run `skills/analytics-baseline/SKILL.md`: Datafast tracking code, key events (page_view, cta_click, form_submit), GSC verification

Read `skills/deploy-ops/SKILL.md` for Cloudflare + Vercel setup.
Read `skills/analytics-baseline/SKILL.md` for event taxonomy.

**Gate:** Page is live at production URL with working analytics.

## G7 — Multi-Agent Audit

Run 6 auditors in parallel:

1. **Responsive Auditor** — Does the page remain persuasive at 390 / 768 / 1280?
2. **Conversion Auditor** — Is the next action obvious and low-friction?
3. **Copy/Simplicity Auditor** — Is any sentence stealing space from the CTA?
4. **Slop Auditor** — Would this immediately read as AI-made?
5. **Legal Compliance Auditor** — Privacy policy linked? Cookie consent present? ToS accessible?
6. **SEO/AEO Auditor** — Meta tags, schema, AI-extractable summary present?

Each auditor returns: **PASS / WARN / FAIL** with evidence.

**Gate:** Zero FAILs required to ship. WARNs are logged for future iteration.

# Verification

Pass only if:
- Page is live at production URL
- Mobile hero renders correctly at 390px
- Primary CTA is clickable and tracked
- Privacy policy and ToS are linked
- Meta tags and schema markup are present
- Zero FAIL verdicts from audit agents
- Analytics events are firing

# Gotchas

- Do not skip G3 (anti-slop). AI-generated landing pages are the #1 trust killer in 2026.
- Do not run full brand OS if you already have one. Use existing brand, don't reinvent.
- Do not add motion/animation before responsive structure passes.
- Do not launch without legal docs. App Store and EU regulators are actively enforcing.
- Word budgets are real: hero headline 6-12 words, support copy 1-2 lines. Shorter converts better.
- Test on a real phone, not just browser resize. Tap targets fail differently on real devices.

# Output Contract

Return:
1. Production URL
2. Brand-to-copy mapping (which brand OS elements became which page sections)
3. Responsive audit results at 3 breakpoints
4. Anti-slop audit verdict
5. SEO/AEO audit verdict
6. Legal compliance checklist (pass/fail per item)
7. Analytics event verification
8. Full audit scorecard with PASS/WARN/FAIL per auditor
9. Fix list for any WARN items
