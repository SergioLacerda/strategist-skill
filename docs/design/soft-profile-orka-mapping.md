# Soft-profile packaging map (ORKA-aligned)

> Source mission: `.analysis/refined/soft_strategist/20260819-strategist-soft/` (`analysis.md`, `proposal.md`, `design.md`, `tasks.md`) — local mission output, not committed to the repo.
> This document materializes Task 1 (and the Task 2 verification note) of that mission's `tasks.md`. It is a design/reference document only — it does not itself authorize building the soft-profile package (Task 3, `implementation_handoff`, remains outside Strategist).

## Why this document exists

The `20260819-strategist-soft` analysis concluded that a CLI-free, Markdown-centered Strategist profile is feasible as an *additive* profile (not a replacement for the hardened Go distribution), but left open how such a package should be physically organized. This document answers that packaging question by mapping the analysis's recommendations onto the organization's ORKA skill-packaging standard, and records a capability disposition matrix so a future implementation mission can proceed without re-deriving either.

## Capability disposition matrix

| Capability | Disposition | Rationale |
|---|---|---|
| Skill entrypoint (`SKILL.md`, `agent-protocol.md`, `active.yaml`) | `simplify` | Condense into ORKA's required `SKILL.md`; move detail into `references/` |
| Phase definitions (`contracts/`, `internal_skills/`, `roles/`, `schemas/`) | `simplify` | Split into one `references/*.md` per phase, ≤200 lines each per ORKA convention |
| Authoring source (`internal/embed/defaults/`) | `preserve` | Remains the hardened profile's single source of truth; the soft profile is a packaging view, not a fork |
| CLI lifecycle/assurance (`cmd/`, `internal/install/`, `compile/`, `check/`, `domain/`, `telemetry/`, `treasure/`, `dojo/`) | `optional CLI` | Stays available as the hardened profile; not required by the soft profile |
| User-facing docs (`README.md`, `QUICKSTART.md`, `docs/onboarding/`) | `preserve` | Already improved by the `20260819-strategist-improvements` mission; the soft-profile `README.md` links to it rather than re-authoring |
| Comparison/precedent artifacts under `.analysis/pending/` | `drop` (as live inputs) | Superseded by this refined package; retained only for historical record |

## ORKA folder map

| ORKA folder | Required? | Soft-profile content |
|---|---|---|
| `SKILL.md` | mandatory | Single concise entrypoint: role lock, entrypoint contract, phase pointers — condensed from the current `.strategist/SKILL.md` |
| `README.md` | optional | Developer-facing onboarding: prerequisites, "who / question answered / artifact produced" table, one-page lifecycle diagram (reuses the front door already built by `20260819-strategist-improvements`) |
| `references/` | optional, ≤200 lines/file, one topic per file | Phase narrative/machine contract content split by phase (`references/discovery.md`, `references/refinement.md`, `references/approval-gate.md`, `references/execution.md`, `references/routing.md`), plus the four non-negotiable controls as `references/governance-controls.md` |
| `scripts/` | optional | Not populated by default — the preflight checklist stays prose-only in `references/` unless a future decision authors an executable helper |
| `templates/` | optional | One template + one example per artifact (analysis/proposal/design/tasks), reusing the shape already validated by this repository's own refined packages |
| `assets/` | optional | Not used — the soft profile has no static/binary resources |

`references/` files are capped at ~200 lines each per the ORKA standard. The current narrative contracts (`00-routing.md` through `11-critical-hit.md`) are each already close to or under that budget individually, so the split is expected to be mostly a rename/relocate operation, not a rewrite.

## Governance controls placement

The four non-negotiable controls that must survive the conversion — role lock, explicit approval gate, bounded write scopes, typed handoff frontmatter — are placed in one file, `references/governance-controls.md`, rather than scattered across phase references, so a reviewer (human or ORKA portal tooling) can audit all four in one place.

## Verification note — SQ-002 (refinement-provider consistency)

**Finding as of 2026-08-20:** no contradiction found. The current `.strategist/active.yaml` resolves `slots.refinement: archivist` directly, with no reference to `openspec-explore`. `strategist check` confirms `refinement → archivist (kind=native_role)` with no external-provider fallback in play. The contradiction originally recorded in the analysis (`KF-05`, sourced from `.strategist/skill.yaml:105-107,192-193` at the time of that discovery) appears to have already been resolved by a prior change to `active.yaml`; this document does not re-verify `skill.yaml`'s own wording, only the resolved runtime configuration. No fix is proposed here — if a residual `skill.yaml` wording inconsistency is found later, it should be raised as its own follow-up rather than expanding this mission's scope.

## What this document does not do

- It does not author the soft-profile package's actual `SKILL.md`, role files, or scripts — that is `implementation_handoff` (Task 3 of the source mission), which requires a separately authorized implementation workflow outside Strategist.
- It does not re-answer feasibility, assurance-parity (`U-02`), or CLI-retirement questions already settled by the source analysis.
- It does not move or edit the closed runbook candidate referenced by the source mission (`RB-20260819-portable-light-client-eval-refinement-escalation.md`) — that is handled separately as a Critical Hit archival move.
