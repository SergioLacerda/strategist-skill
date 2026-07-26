# Runbook — Skill Deep-Analysis Workflow (4 axes, 6 passes)

Systematic analysis of a governed skill corpus (contracts + schemas + runtime code)
producing a ranked, deduplicated improvement backlog. First executed 2026-07-26 against
the strategist skill itself; written to be repeatable by a Strategist mission
(`discovery_subtype: evaluation`/`diagnostic`) or by an agent following it directly.

## Trigger

- Periodic health review of the skill corpus (suggested: after every N missions or
  before a structural refactor).
- After a cluster of drift/consistency incidents.
- Onboarding review of an unfamiliar skill with the same shape (docs corpus + machine
  contracts + runtime implementation).

## Axes

1. Main flow — simplification & reuse opportunities
2. Main flow — defects, ambiguities, consistency gaps
3. Mechanism synergy — sub-ability interactions, missing mechanics
4. World-class engineering criteria — scored rubric

## Structure

Six passes, each one session-sized, each producing one findings artifact:

| Pass | Runbook | Output artifact |
|---|---|---|
| 0 — Inventory & metrics | `deep-analysis-pass-0-inventory.md` | `00-inventory.md` |
| 1 — Simplification & reuse | `deep-analysis-pass-1-simplification.md` | `01-simplification.md` |
| 2 — Defects (adversarial) | `deep-analysis-pass-2-defects.md` | `02-defects.md` |
| 3 — Mechanism synergy | `deep-analysis-pass-3-synergy.md` | `03-synergy.md` |
| 4 — Engineering rubric | `deep-analysis-pass-4-engineering.md` | `04-engineering.md` |
| 5 — Consolidation & plan | `deep-analysis-pass-5-consolidation.md` | `refined/<slug>/analysis.md` + `tasks.md` |

Order: 0 first (cheap, mechanical, feeds evidence to all others); 1–4 independent, any
order, parallelizable across sessions; 5 requires all four.

## Artifact contract

- Resolve `base_path` from `.strategist/active.yaml` — never hardcode the workspace dir.
- Pass 0–4 findings: `<base_path>/pending/<analysis-slug>/0N-<name>.md`.
- Pass 5 package: `<base_path>/refined/<date>-<slug>/analysis.md` + `tasks.md`,
  presented at an approval gate.
- Findings language follows `active.language.docs`.

## Stop Conditions (all passes)

- **Analysis-only.** No source mutation, no git state changes. Fix proposals are
  classified `documentation_target` vs `implementation_handoff` in Pass 5; handoff items
  are executed only by a separately authorized task after gate acceptance.
- **Freeze a baseline.** Record branch + commit in every artifact header; parallel-session
  changes are recorded, not treated as anomalies.
- **Every finding carries:** stable ID, location (`file:line` where possible), evidence,
  concrete failure scenario or proposal, effort estimate.
- **Errata are appended, not silently rewritten.** If a later pass invalidates an earlier
  claim, the later artifact records an erratum naming the earlier finding.
- **Dedupe before proposing.** Check the existing backlog (`<base_path>/todo/`) and link
  instead of restating.

## Decision Point

The workflow ends at the Pass 5 approval gate. Done means: 5 findings artifacts +
consolidated refined package exist; zero unlinked duplicates against the pre-existing
backlog; gate presented and the decision recorded in the package frontmatter.
Acceptance never authorizes executing `implementation_handoff` items — those become
separate tasks.

## Reference

- Pass runbooks: `deep-analysis-pass-0-inventory.md` … `deep-analysis-pass-5-consolidation.md`
- First execution (worked example): `.analysis/pending/strategist-deep-analysis/` +
  `.analysis/refined/2026-07-26-strategist-deep-analysis/`
- Procedural-shape sibling: `verifying-implemented-demands.md`
