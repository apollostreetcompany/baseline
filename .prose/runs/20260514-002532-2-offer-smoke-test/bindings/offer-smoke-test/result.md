# Offer Smoke Test Output: Baseline.ai

1. Chosen test type: Stripe checkout / paid pilot fake-door hybrid. Stronger than waitlist; honest because checkout fails closed until configured.

2. Page structure and CTA: Live landing page leads with local-first drift monitor, visual dashboard, install MCP CTA, and Start Pro CTA.

3. Traffic / outreach plan: DM 10 OpenClaw/Hermes power users, 5 agency owners, and 5 CTO/founder-CTOs. Ask them to run the MCP and click Start Pro if they want cloud history/alerts.

4. Measured actions available now: page view, CTA click via /api/events, checkout attempted via /api/checkout, cloud run ingest via /api/runs.

5. Objections and confusion notes: payment is the blocker; Stripe secrets or payment links are not configured. Also clarify that raw prompts are not exported.

6. Verdict: REFINE before paid launch, PASS for dogfood.

7. Next route: configure Stripe payment links, send direct outreach, measure checkout attempts and replies for 3-7 days.
