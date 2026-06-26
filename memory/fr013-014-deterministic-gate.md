---
name: fr013-014-deterministic-gate
description: When the first AI/insight endpoint lands (SPEC-102+), promote the FR-013/FR-014 product guards from subjective review to a deterministic gate
metadata:
  type: project
---

The binding product guards **FR-013 (explainability)** and **FR-014 (non-advice)** are
today only enforced subjectively — by prose in CLAUDE.md and by the `hexagonal-reviewer`
subagent (§2), which is opt-in and judgment-based. There is **no deterministic gate** yet,
because no endpoint produces AI/insight output (SPEC-005 shipped only the `Insighter`
*port*; SPEC-101 is the investor profile).

**Trigger:** when the first endpoint that emits an LLM insight/score/suggestion lands
(expected SPEC-102 portfolio / SPEC-103 dashboard), promote these guards to a hard,
machine-checkable gate — the *product harness* analog of the dev-harness
`block-layering` hook (see [[harness-engineering-notes]] if created).

**How to apply:** enforce at the `Insighter` seam, not in prose —
- reject any insight whose explanation field is empty/missing (FR-013);
- scan output for specific buy/sell orders, tickers-to-buy, quantities, or price targets
  and block + require a non-advice disclaimer (FR-014);
- keep facts computed (Fact Builder), never generated.

This is the most on-theme next harness step and the user's deliberate learning goal: the
*two harnesses* — dev harness (around Claude) vs product harness (around the Insighter).
Links [[yieldforge-project]].
