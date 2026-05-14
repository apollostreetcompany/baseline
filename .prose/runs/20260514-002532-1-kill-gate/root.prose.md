---
name: kill-gate
description: Score a startup idea on solo-founder feasibility before committing build resources. Kills bad ideas early.
entry-stage: G0
target-fidelity: F1 (decision only)
use-when:
  - A new product idea needs go/no-go before any building starts.
  - You want to compare 2-5 ideas and pick the best one to pursue.
  - An idea feels exciting but you need a reality check on solo execution.
avoid-when:
  - The product is already built and generating revenue.
  - The decision is about a feature within an existing product, not a new venture.
  - You have a full team and traditional funding — this gate is calibrated for solo/micro founders.
skills:
  - ce:brainstorm (idea clarification)
  - predictor (calibrated scoring)
  - competitor-analysis.prose (market check)
  - app-intel (market size signals)
audit-agents:
  - Niche Viability Auditor
  - Solo Execution Auditor
  - Revenue Timeline Auditor
inputs:
  required:
    - Idea description (1-3 paragraphs)
    - Target user and their current painful alternative
    - How you imagine making money
  optional:
    - Existing skills or assets you bring
    - Time budget (hours/week)
    - Token/compute budget
outputs:
  - Kill gate scorecard (go / conditional-go / kill)
  - Niche analysis with defensibility rating
  - Solo execution feasibility assessment
  - Recommended next recipe if go
---

# Goal

Decide in under 30 minutes whether an idea deserves factory resources. The bar is: can a solo founder with high token budget, design sensibility, and agentic tooling WIN in this space without being consumed?

# Core Philosophy

**Niches == riches.** The default winning strategy for a solo founder is:
- Find a niche too small for funded teams to care about
- Have a better product insight than anyone currently serving it
- Ship faster than anyone expects using agentic leverage

Ideas that require competing head-on with well-funded incumbents on their core feature get killed unless you have a genuine 10x insight.

# Workflow

## G0 Intake — Clarify the idea

Use `ce:brainstorm` principles to force clarity:

1. **What is it?** One sentence. If it takes a paragraph, the idea isn't clear yet.
2. **Who is the user?** Name a specific person, not a demographic.
3. **What do they do today?** The painful current alternative.
4. **Why would they switch?** The trigger moment.
5. **How do you make money?** Revenue model in one line.

## G1 Score — Seven Kill Dimensions

Score each dimension 1-5. Total possible: 35.

### 1. Niche Clarity (1-5)
- 1 = "everyone" / mass market
- 3 = clear segment but crowded
- 5 = specific underserved niche you can name 100 people in

### 2. Solo Execution Feasibility (1-5)
- 1 = requires a team of 5+ or regulatory approval
- 3 = one person can build MVP but growth needs help
- 5 = one person with agentic tools can build, launch, and grow

### 3. Incumbent Vulnerability (1-5)
- 1 = Google/Apple/Amazon core feature
- 3 = funded startups exist but none dominate
- 5 = incumbents are lazy, expensive, or structurally unable to serve this niche

### 4. Revenue Speed (1-5)
- 1 = 2+ years to first dollar
- 3 = 6 months to revenue with effort
- 5 = can charge from day one / pre-sell

### 5. Agentic Leverage (1-5)
- 1 = AI/agents don't help much (physical product, regulated industry)
- 3 = agents help with marketing/content but not the core product
- 5 = the product IS agentic leverage (AI-native, content pipeline, automation)

### 6. Insight Advantage (1-5)
- 1 = no unique insight, just execution
- 3 = you understand the user better than current options
- 5 = you have proprietary data, workflow knowledge, or domain expertise others lack

### 7. Personal Energy (1-5)
- 1 = "sounds profitable but I don't care"
- 3 = interested enough to work on it for 6 months
- 5 = obsessed, would build it even without money

## G2 Research — Validate or Invalidate

Spawn parallel sub-agents:

**Agent 1: Market Check**
- How many competitors exist? (use competitor-analysis or web search)
- What do they charge?
- What do users complain about in reviews?
- Is the market growing or shrinking?

**Agent 2: Niche Depth**
- Can you find 3 specific communities where target users gather?
- Are they actively complaining about the problem?
- Would 100 of them pay $X/month?

**Agent 3: Build Estimate**
- What's the minimum viable product?
- How many screens / endpoints / integrations?
- Can it ship in 2 weeks with agentic tools?

## G3 Decision — Kill Gate Verdict

### KILL (score < 18 OR any dimension = 1)
Stop. Do not proceed. Log the idea and why it died in meta-learning.

### CONDITIONAL GO (score 18-25)
Proceed to one more validation step before building:
- If niche clarity is low → run Tibo/market research recipe first
- If execution is uncertain → run iOS sprint timeline estimate
- If revenue is slow → run monetization strategy check

### GO (score 26-35)
Proceed to factory. Recommended next recipe based on highest-leverage gap:
- Weak brand → recipe 06 (brand messaging)
- Weak market fit → recipe 03 (brainstorm + Tibo)
- Ready to build → recipe 08 (iOS sprint)
- Need landing page first → recipe 07 (brand → landing)

## G4 Calibration — Store for Learning

Record the prediction:
- Idea name
- Kill gate score and breakdown
- Decision (kill / conditional / go)
- Confidence level (%)
- Date
- Predicted outcome at 90 days

This feeds into recipe 13 (meta-learning across projects) for calibration.

# Verification

Pass only if:
- All 7 dimensions are scored with evidence, not vibes
- At least one research agent returned real market data
- The decision is GO, CONDITIONAL, or KILL — not "maybe"
- The prediction is stored for future calibration

# Gotchas

- **Excitement is not a score.** Personal energy (dim 7) is only 1 of 7 dimensions. A 5 there doesn't save a 1 elsewhere.
- **"No competitors" is a red flag**, not a green one. It usually means no market.
- **Don't score agentic leverage high just because you have agents.** Score it high only if agents give you a structural advantage the incumbent can't easily copy.
- **The kill gate is not permanent.** Ideas can be re-scored when circumstances change (new tool, new insight, market shift).
- **Conditional go has a time limit.** If the validation step isn't done in 1 week, treat it as a kill.

# Output Contract

Return:
1. Idea summary (1 sentence)
2. Seven-dimension scorecard with evidence
3. Total score and verdict (KILL / CONDITIONAL GO / GO)
4. If GO: recommended next factory recipe
5. If CONDITIONAL: specific validation step and deadline
6. Calibration record for meta-learning storage
