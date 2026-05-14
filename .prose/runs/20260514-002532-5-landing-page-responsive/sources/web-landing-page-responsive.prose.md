---
name: landing-page-responsive
description: Make a landing page work on mobile and desktop without losing hierarchy, conversion clarity, or brand character.
platform: web
entry-stage: G0
target-fidelity: F3 for review, F5 for launch
use-when:
  - A landing page looks good on desktop but collapses on mobile.
  - The page feels persuasive in one viewport and messy in another.
  - You need a structured way to check hero, CTA, copy, and layout across breakpoints.
avoid-when:
  - The surface is a logged-in product UI or dashboard.
  - The problem is mostly backend checkout logic; use the integration recipe instead.
skills:
  - frontend-design
  - ui-arrange
  - adapt
  - ui-clarify
  - distill
  - ui-critique
  - ui-audit
  - fixing-accessibility
  - fixing-web-metadata
  - baseline-ui
audit-agents:
  - Responsive Auditor
  - Conversion Auditor
  - Copy/Simplicity Auditor
  - Slop Auditor
inputs:
  required:
    - Current page URL or screenshots at 390px and desktop
    - Primary conversion goal
    - Target audience and offer
  optional:
    - Existing analytics or drop-off notes
    - SEO / social preview requirements
outputs:
  - Prioritized issue list by gate
  - Responsive redesign notes or implementation plan
  - Multi-agent audit verdict
---

# Goal

Preserve persuasion and visual character while making the landing page feel intentional at 390px, 768px, and desktop.

# Input state capture

Collect:
- current screenshots at a minimum of **390px**, **768px**, and **1280px**
- the exact primary CTA
- the offer, proof points, and target audience
- the one visual trait that gives the page character
- the 3 biggest complaints, if known

# Desired result capture

Success at F3-F5 means:
- the value proposition is visible in the first screen on mobile
- the primary CTA is obvious within 2 seconds
- no section feels like a desktop layout simply squeezed smaller
- copy does not wrap awkwardly or dominate the layout
- the page still feels like *this* brand, not a generic responsive template

# Workflow

## G0 Intake
Use `frontend-design` to restate the audience, offer, and anti-goals. Set the target to F3 unless shipping.

## G1 Direction
Run `ui-critique` lightly first to detect whether the page already looks AI-generated. Name the signature to preserve and the slop to remove.

**Gate check**
- one sentence value proposition exists
- one primary CTA exists
- one signature trait is named

## G2 Structure
Use `ui-arrange` and `distill` together.

Check:
- hero order on mobile: headline → proof → CTA → supporting media
- no more than one primary CTA per screenful
- sections are grouped by user questions, not by design tropes
- decorative cards, icon rows, and filler stats are cut unless they support conversion

## G3 Platform fit
Use `adapt`.

Required responsive checks:
- 390px: headline line count, CTA visibility, media crop, proof placement
- 768px: section rhythm, stack order, scroll fatigue, CTA repetition
- 1280px: line length, whitespace balance, dead zones, oversized media

Do not accept simple scale-down. Recompose sections when needed.

## G4 Copy + simplicity
Use `ui-clarify` and `distill`.

Word budgets:
- hero headline: target 6-12 words
- hero support copy: target 1-2 short lines
- feature blocks: target 1 sentence each before expansion
- proof sections: say what changed for the user, not abstract praise

## G5 Motion + delight
Only after responsive hierarchy passes.

If motion is used:
- hero reveal should not delay CTA readability
- motion should help sequence attention, not add spectacle
- skip heavy animation on dense benefit sections

## G6 Hardening + release checks
Run `fixing-accessibility`, `baseline-ui`, `fixing-web-metadata`, and optionally `ui-audit`.

Check:
- tap targets and focus order
- no overflow or horizontal scroll
- metadata, share cards, page title, and description reflect the actual offer
- loading performance on mobile is acceptable

## G7 Multi-agent audit
Run these auditors:
- **Responsive Auditor**: Does the page remain persuasive at 390 / 768 / 1280?
- **Conversion Auditor**: Is the next action obvious and low-friction?
- **Copy/Simplicity Auditor**: Is any sentence stealing space from the CTA?
- **Slop Auditor**: Would this immediately read as AI-made?

# Verification

Pass only if:
- mobile hero fits without broken wrapping or clipped CTA
- desktop layout does not create long unreadable lines
- no section is kept solely because it looks “complete”
- the share/SEO metadata matches the launch message

# Gotchas

- Do not start with animation. Responsive hierarchy comes first.
- Do not keep a feature grid just because it fills space.
- Do not fix desktop and call mobile “good enough.” Mobile is usually the harsher test.

# Output contract

Return:
1. current-state failures by viewport
2. redesign plan by gate
3. required skill calls
4. final audit verdict with blockers
