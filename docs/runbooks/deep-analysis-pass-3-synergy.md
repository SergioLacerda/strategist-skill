# Runbook — Deep Analysis Pass 3: Mechanism Synergy & Sub-Abilities

Map how the skill's mechanics (sub-abilities) interact, find dead links, and propose new
mechanics **in the same design language** as the existing ones.

## Trigger

Pass 0 inventory complete. Runs independently of Passes 1–2 (cross-references welcome
when they exist).

## Steps

### 1. Mechanic inventory

One row per mechanic: **name | kind (route/radar/routine/knowledge/safety) | fires at |
gated by | output**. Sources: machine contracts, internal skill manifests, narrative
contracts, offline tooling (e.g. dojo scenarios).

### 2. Interaction matrix

Mechanic × mechanic, each cell: `✔` reinforces · `○` independent · `✖` broken/dead.
For every `✖`, record evidence. The two highest-yield probes:

- **Write-without-reader:** a mechanic records something (e.g. a backlog status, a
  declined-item list) — grep for consumers. Zero readers = dead link.
  ```sh
  grep -rn '<recorded-token>' <corpus> internal/ tests/   # writers vs readers
  ```
- **Signal-not-at-decision-point:** a quality/priority signal exists (critic score,
  staleness flag) — check whether it reaches the human gate where the decision is made.

Also verify offline-validation coverage: which mechanics have dojo/test scenarios and
which don't.

### 3. Identify keep-patterns

Name the best-designed mechanic pairs (clear ownership split, evidence-gated, bounded
triggers) — new proposals must copy their doctrine, and refactors must not touch them
casually.

### 4. New sub-ability proposals

Template per proposal (all fields required):

- **Trigger** — where in the pipeline it fires;
- **Behavior** — bounded description; advisory vs acting;
- **Touchpoints** — contracts/schemas/code it extends (reuse existing machinery first);
- **Composes with** — which existing mechanics/links it fixes or feeds;
- **Must NOT bypass** — gates and doctrines it preserves (this field is the design
  language: every mechanic surfaces candidates, humans decide).

**Dedupe duty:** before proposing, check the backlog and CLI namespace —
a proposal may already exist as a captured entry (link it) or collide with an existing
command name (`grep -rhoE 'Use: +"[a-z-]+' cmd/`).

## Decision Point

Done when `<base_path>/pending/<slug>/03-synergy.md` exists: inventory + matrix + dead
links `Y<n>` + proposals `P<n>` + priority view separating defect-fixes (route to Pass
2/5) from genuinely new mechanics.

## Stop Conditions

- No proposal may create an ungated write path — if the "Must NOT bypass" field is hard
  to fill, the proposal is wrong.
- Proposals that feed a pending architectural decision are inputs to that decision, not
  parallel designs — say so explicitly.
- A proposal without the dedupe check (backlog + CLI namespace) is not done.

## Reference

- `deep-analysis-workflow.md` (master); `deep-analysis-pass-0-inventory.md` (input).
- Worked example: `.analysis/pending/strategist-deep-analysis/03-synergy.md` (2026-07-26).
