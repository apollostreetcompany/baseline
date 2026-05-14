---
name: offer-smoke-test
description: Prove willingness to act before a full build using a waitlist, deposit, payment link, fake door, or concierge offer.
entry-stage: G0
target-fidelity: F1-F2
use-when:
  - Customer Discovery passed and you need action or payment signal.
  - A kill-gate decision is blocked on willingness-to-pay evidence.
  - You want to validate demand with the fastest live market test.
avoid-when:
  - You already have strong paid demand or a stable product funnel.
  - You are trying to optimize an existing production landing page.
  - You want a full brand site or complete product build.
skills:
  - skills/brand-os-studio/SKILL.md (light mode only)
  - recipes/web-landing-page-responsive.prose.md
  - skills/revenue-plumber/SKILL.md
  - skills/deploy-ops/SKILL.md
  - skills/analytics-baseline/SKILL.md
  - skills/legal-shield/SKILL.md
audit-agents:
  - Offer Clarity Auditor
  - Conversion Auditor
  - False-Positive Auditor
inputs:
  required:
    - target user
    - one clear promise
    - one primary CTA
  optional:
    - interview quotes
    - price hypothesis
    - outreach list or traffic source
outputs:
  - live smoke-test page
  - CTA wiring (waitlist / deposit / payment link / booking)
  - basic measurement
  - pass / refine / kill verdict
---

# Goal

Get proof that people will do something meaningful:
- join
- book
- deposit
- pay
- reply
- opt into a concierge pilot

This recipe exists to prevent unnecessary builds.

# Allowed test types

Pick one. Do not mix five things at once.

1. **Waitlist** — lowest friction, weakest signal
2. **Booked call / pilot application** — stronger signal for higher-ticket or service-like offers
3. **Deposit / preorder / payment link** — strongest pre-build signal
4. **Fake door** — valid only if you clearly capture intent and follow up honestly
5. **Concierge offer** — manually deliver the result before software exists

Prefer the strongest honest test your project can support.

# Timebox

- 2-6 hours to launch
- 3-7 days to collect signal
- decide immediately after the window closes

# Workflow

## G0 Choose the proof event

Define the one action that counts.

Examples:
- waitlist signup
- Stripe payment link checkout started/completed
- application submitted
- “book intro call” completed
- reply to targeted outreach

If you cannot name the proof event, you are not ready to run the test.

## G1 Build the minimum page

The page should only answer:
1. Who is this for?
2. What painful outcome gets fixed?
3. Why this angle?
4. What should the visitor do now?

Keep it minimal:
- one headline
- one supporting block
- one proof point
- one CTA
- no feature grid unless it materially changes conversion

Do **not** run full brand OS here. Light copy only.

## G2 Wire the action

Use the simplest honest mechanism:

- waitlist form
- Stripe Payment Link / buy button
- booking link
- direct email reply
- application form

Instrument only what matters:
- page view
- CTA click
- completion event

## G3 Push qualified traffic

Use targeted traffic, not random vanity traffic.

Preferred sources:
- direct outreach from discovery interviews
- niche communities
- existing audience
- partner/influencer intros
- relevant social posts

A small qualified sample beats broad low-intent traffic.

## G4 Read the result

Look at:
- conversion rate on the primary CTA
- quality of signups or replies
- objections and friction
- whether users are confused about the promise

Then decide:

### PASS
Use when the action rate is meaningfully positive for the traffic quality and the replies show real intent.

Next routes:
- `skills/brand-os-studio/SKILL.md`
- the relevant build recipe: `recipes/21-full-brand-os-production-landing.prose.md` or `recipes/22-ios-release-lane.prose.md`

### REFINE
Use when people are interested but the page, pricing, or CTA is muddy.

Next route:
- tighten promise / price / proof and rerun once

### KILL
Use when the traffic is qualified but people still do not act.

Do not rescue a weak offer with more design or more code.

# Verification

Pass only if:
- the smoke test is live
- the primary action is measurable
- traffic or outreach is qualified
- you produce a hard verdict after the window closes

# Gotchas

- A pretty page with no action is not validation.
- Waitlists are weaker than deposits or payments.
- Unqualified traffic creates false negatives.
- Friend praise creates false positives.
- Do not confuse CTR with willingness to pay.

# Output contract

Return:
1. chosen test type
2. page structure and CTA
3. traffic / outreach plan
4. measured actions
5. objections and confusion notes
6. pass / refine / kill verdict
7. recommended next route
