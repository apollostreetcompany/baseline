---
name: anti-slop-cleanup
description: Detect and remove generic AI-generated visual and copy patterns while preserving the product's actual signature.
platform: web
entry-stage: G1
target-fidelity: F3-F5
use-when:
  - The UI feels templated, noisy, awkward, or over-decorated.
  - You see generic icons, random cards, gradients, filler metrics, or bad grammar.
  - A page “looks designed” but not specific to the product.
avoid-when:
  - The main problem is a broken integration or navigation handoff.
  - The surface needs a new concept, not cleanup.
skills:
  - ui-critique
  - distill
  - ui-clarify
  - ui-quieter
  - ui-normalize
  - baseline-ui
  - ui-audit
audit-agents:
  - Slop Auditor
  - Character Auditor
  - Copy Auditor
inputs:
  required:
    - Screenshots or page/component files
    - Brand personality or product signature
    - A statement of what must stay
  optional:
    - A list of things the team already hates
outputs:
  - Slop inventory
  - Keep / remove / replace decisions
  - Final pass/fail audit
---

# Goal

Remove generic, noisy, or obviously AI-made design choices without sterilizing the product.

# Input state capture

Capture:
- three screenshots or components that feel most “AI-made”
- one sentence on the brand/product personality
- one sentence on what character or weirdness is worth preserving

# Desired result capture

The surface should feel more specific, more readable, and more intentional — not simply flatter.

# Workflow

## G0 Intake
List the exact evidence of slop. Name specific patterns, not vibes.

Examples:
- identical icon cards repeated for no reason
- decorative gradients that communicate nothing
- misaligned boxes or inconsistent radii
- filler headings or support copy that restate the obvious
- generic dashboards-in-disguise layouts

## G1 Direction
Use `ui-critique` first.

Create three lists:
- **keep** — distinctive choices worth preserving
- **remove** — generic choices that harm trust
- **replace** — weak choices that need a more product-specific alternative

## G2 Structure
Run `distill`.

Remove first:
- redundant sections
- repeated proof points
- unnecessary containers or nested cards
- icon + heading + paragraph patterns with no real information gain

## G3 Platform fit
Run `ui-normalize` or `baseline-ui` as needed.

Check for:
- inconsistent spacing scale
- inconsistent border / surface treatment
- type hierarchy drift
- one-off colors or radii

## G4 Copy + simplicity
Run `ui-clarify`.

Rewrite:
- headings that say nothing
- support copy that exceeds the visual space
- button text like “Learn More,” “Click Here,” or “Submit” when a real action is known
- grammar or phrasing that breaks trust

## G5 Tone calibration
If the surface is loud rather than cluttered, run `ui-quieter` after `distill`.

Only use `ui-bolder` if the page becomes too safe after cleanup.

## G6 Hardening
Run `ui-audit` once cleanup is complete.

## G7 Multi-agent audit
- **Slop Auditor**: What would make someone instantly say “AI made this”?
- **Character Auditor**: Did cleanup preserve the signature?
- **Copy Auditor**: Did words become sharper and shorter?

# Verification

Pass only if:
- every kept element can justify itself
- the page looks more specific to the product than to a trend
- there are fewer surface types, not just softer ones
- copy reads as deliberate, not generated filler

# Gotchas

- Do not erase all personality. The goal is specificity, not beige minimalism.
- Do not swap one generic trend for another.
- Do not call it fixed if you only changed colors but kept the same clutter.

# Output contract

Return:
1. slop inventory
2. keep/remove/replace table
3. final set of follow-up recipes if the page still fails
