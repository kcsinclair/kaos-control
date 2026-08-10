---
title: Record a cost basis per agent run so local models report savings, not spend
type: idea
status: draft
lineage: agent-cost-basis-and-savings
priority: medium
labels:
    - agent
    - reports
    - cost
assignees:
    - role: product-owner
      who: agent
---

# Record a cost basis per agent run so local models report savings, not spend

## Context

The agent usage report currently treats every run's `total_cost_usd` as money
spent. That value comes from the Claude Code stream-json `result` event, which
applies **Anthropic's price table to the token counts regardless of which
endpoint actually served the request**. So a `claude-env` run against the local
Ollama box on `leia` is recorded as if Anthropic had billed for it.

Measured on this project's own `agent_runs` table (2026-07-17):

| Month | Local runs | Reported "cost" |
|---|---|---|
| 2026-06 | 6 | $92.76 |
| 2026-07 | 2 | $28.30 |

That is **$121 of reported spend that never left the building**. One run — a
`gemma4:26b-a4b-it-qat` `test-developer` that ran 240 minutes over 94 turns — is
recorded at $58.02 and is the single most expensive run in the table. It cost
nothing.

For the same window, the real Anthropic-billed agent spend was:

| Month | Claude runs | Cost at API rates |
|---|---|---|
| 2026-04 | 101 | $77.47 |
| 2026-05 | 486 | $432.19 |
| 2026-06 | 80 | $129.02 |
| 2026-07 (to 17th) | 28 | $62.09 |

So the report currently overstates recent cost by roughly 50%.

## The problem to solve

The report conflates **money spent** with **money not spent**. Both numbers are
worth having, and the second one is arguably the more interesting: "your local
box saved you $121 in two months" is a direct argument for the local-model
investment, and it is exactly the number needed to make subscription-vs-metered
decisions. Today it is silently laundered into the spend column, where it
actively misleads that decision.

The fix is not to zero local runs out. It is to **classify** them.

## The design decision: `claude-env` is not a synonym for "free"

The obvious implementation — "driver is `claude-code-cli` ⇒ billed, everything
else ⇒ saved" — is wrong, and will become wrong the day an OpenRouter agent is
configured. `claude-env` only says *"an Anthropic-compatible endpoint that isn't
Anthropic"*. Where it points determines the economics:

- `base_url: http://leia.packsin.com:11434` (Ollama) → **saved**, no money moves.
- `base_url: https://openrouter.ai/api` → **metered**, real money, just not
  Anthropic's — and priced by *OpenRouter's* table, not the one in
  [internal/reports/pricing.go](../../internal/reports/pricing.go).

So there are three bases, not two: **billed** (Anthropic), **metered**
(third-party paid), **saved** (self-hosted). Inferring these from the hostname
is unreliable — `leia.packsin.com` looks like a public domain and is not
RFC1918-detectable by name. It has to be declared.

## The other design decision: record it on the run, not derive it at report time

The basis must be **stamped on the `agent_runs` row when the run is recorded**,
not looked up from `config.yaml` when the report is rendered. Config is mutable:
repoint an agent from `leia` to OpenRouter and every historical run of that
agent would silently re-classify, rewriting the past. This is the same reason
`model` is already denormalised onto the run row rather than joined from config.

`Run.Driver` and `Run.BaseURL` are already carried through to the supervisor
([internal/agent/agent.go](../../internal/agent/agent.go)), so the value is
available at insert time. The schema migration is a one-liner alongside the
existing block at
[internal/index/index.go L1596-1605](../../internal/index/index.go#L1596):

```go
_, _ = idx.db.Exec(`ALTER TABLE agent_runs ADD COLUMN cost_basis TEXT`)
```

## Sketch

1. **Config** — a per-agent `cost_basis:` field (`anthropic` | `metered` |
   `local`), defaulted by driver: `claude-code-cli` / `claude-mediated` →
   `anthropic`; `ollama` → `local`; `claude-env` → **required**, since it is
   genuinely ambiguous (validate it the way `base_url` / `auth_token` /
   `model` are already validated at
   [internal/config/config.go L736-748](../../internal/config/config.go#L736)).
2. **Run + index** — carry the resolved basis on `Run`, stamp it into
   `agent_runs.cost_basis` on insert.
3. **Reports** — split the aggregate: `total_cost_usd` (billed + metered) and a
   new `total_saved_usd` (local). Per-model and per-agent breakdowns get the
   same split; CSV export gains the column.
4. **UI** — a "saved" tile / column beside spend on the usage report.
5. **Backfill** — existing rows have no basis. Either leave them `unknown` and
   exclude from both columns, or one-shot classify by model name (`claude*` →
   anthropic, else local), which is accurate for every run recorded so far.

## Open questions (to think about)

- **Is `metered` worth building now?** There are no third-party paid agents
  today. Ship `anthropic` / `local` only and add `metered` when OpenRouter is
  actually configured, or build all three up front so the taxonomy doesn't
  churn later?
- **Pricing for metered endpoints.** `pricing.go` only knows the Claude
  families and `splitCost` returns no split for unknown models. An OpenRouter
  run's recorded `total_cost_usd` would still be Anthropic-priced and therefore
  wrong in a *third* way — neither spend nor saving, just fiction. Does
  `metered` need its own price table, or does it just report tokens and no cost
  until one exists?
- **What is "saved" actually worth?** A local run priced at Anthropic's Opus
  rates claims a saving you'd never have paid — you'd have used Sonnet, or not
  run it at all. Is the honest number "what this would have cost on the model
  you'd otherwise have used", and if so, which model? Or is the naive
  same-token-count figure good enough to be useful?
- **Runs with no metrics.** 70 of 775 runs have `metrics_available = 0` (the
  `gemini-2.5-flash` runs report zero tokens entirely). Do they belong in a
  third "unmeasured" bucket so the totals stay honest about coverage?
- **Backfill vs. leave alone.** Rewriting history by inference is exactly what
  this idea argues against — but leaving 775 rows unclassified means the report
  stays wrong for everything before the change. Which is worse?

## Related

- [agent-run-branch-isolation.md](agent-run-branch-isolation.md) — the 240-minute
  $58 runaway that motivates this is the same class of failure branch isolation
  is meant to contain.
