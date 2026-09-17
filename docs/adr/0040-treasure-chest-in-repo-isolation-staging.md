# ADR-0040 — Treasure Chest In-Repo Isolation Staging (Jewelcrafter Role)

**Status:** Proposed
**Date:** 2026-09-15
**Mission:** `20260915-treasure-chest-root-path-isolation`
**Supersedes:** `docs/adr/0032-external-skill-cli-embedding-and-treasure-chest-ownership.md` — staging-plan section only ("Staged Treasure Chest removal"); ADR-0032's Decision, Alternatives, and Consequences remain in force unchanged.

## Context

ADR-0032 (Accepted, 2026-09-13) decided Treasure Chest becomes an
independently-versioned external repository, consumed via a message-oriented
CLI Extension API and a committed embedded-skill lock, with `strategist
treasure-chest ...` staying as a stable compatibility namespace. None of its
seven implementation tasks (T1–T7) have executed; `internal/treasure/**` and
`internal/treasurecli/**` remain fully intact in the current codebase.

This ADR adds a staging stage in front of ADR-0032's existing plan: relocate
Treasure Chest into a root-level, in-repo path first, mediated by a new
non-pipeline Strategist role, as an explicit "baby step" toward ADR-0032's
external-repository target — not a replacement for it.

## Decision

1. **Relocate Treasure Chest to `treasure-chest/` at the repository root**,
   within the existing single Go module (no separate `go.mod` at this stage).
   This includes `internal/treasure/**`, `internal/treasurecli/**`, and the
   treasure-chest-specific domain types currently inside the shared
   `internal/domain` package (`chest_grade.go`, `jewel_grade.go`,
   `potion_grade.go`, `promotion_packet.go`) — these move into
   `treasure-chest/` (e.g. `treasure-chest/domain/`) rather than staying
   behind as a shared dependency.
2. **Introduce Jewelcrafter as a new, non-pipeline internal Strategist
   skill** (`internal_skills/jewelcrafter/{skill.yaml,SKILL.md}` —
   corrected 2026-09-15; see addendum below for why there is no
   `roles/jewelcrafter.yaml`). `skill.yaml`'s `slots:` map is unchanged
   (`discovery`/`refinement`/`execution` only) — Jewelcrafter is never a
   fourth pipeline slot. Jewelcrafter's scope: map and organize the offline
   document structure (ADRs, runbooks, docs); pre-mine and organize items
   for discovery/refinement to consult; manage TTL/lifecycle of the
   artifacts it creates; own `.strategist/.data-offline/` as its runtime
   domain. Ranger's `consult_treasure_chests` ability (and Archivist's
   `reuse_search_cache`) call Jewelcrafter as a sub-routine rather than
   importing `treasure-chest/` directly.
3. **The external-repository infrastructure ADR-0032 already specified
   (CLI Extension API, embedded-skill lock entry, pre-build ingestion) is
   explicitly not built by this stage.** It remains ADR-0032's own,
   unchanged, unexecuted T1–T7. This ADR's only relationship to that work is
   that Jewelcrafter is designed as the seam where, later, only its internals
   change to call an external skill/connector instead of importing
   `treasure-chest/` in-process — its callers (Ranger, Archivist) do not
   change at that point.
4. **The `jewelStatusLegacyActive` migration guard (ADR-0012) and its
   error message must remain valid after the move.** The error text names
   `strategist treasure-chest items migrate-status`; after relocation, this
   command must still resolve, now via Jewelcrafter's mediation instead of
   direct core registration — this is a required verification step, not an
   assumption.
5. **No separate Go module yet.** `treasure-chest/` stays part of the
   existing root module. The isolation proof for this stage rests on the
   Jewelcrafter role boundary (no other package may import `treasure-chest/`
   directly), not on a module boundary. A future decision may add a separate
   `go.mod` when the external-repository stage (ADR-0032) actually begins.

## Staged plan (supersedes ADR-0032's "Staged Treasure Chest removal" only)

0. **(New stage, this ADR)** Relocate Treasure Chest to `treasure-chest/`
   in-repo; introduce Jewelcrafter; verify the migration-guard error path;
   no separate module yet.
1. Define and publish the external package/API and parity fixtures (ADR-0032, unchanged).
2. Add pre-build ingestion, embedded catalog and extension routing while core
   Treasure Chest remains available behind a migration/compatibility boundary
   (ADR-0032, unchanged).
3. Validate external skill command parity and import existing workspace data (ADR-0032, unchanged).
4. Switch the namespace to the embedded skill implementation (ADR-0032, unchanged).
5. Remove core implementation/defaults after the deprecation window (ADR-0032, unchanged).

## Alternatives considered

### Treat this as a permanent target, superseding ADR-0032 entirely

Rejected. The user's stated intention is to later import Treasure Chest from
an external repository — matching ADR-0032's target, not replacing it. An
in-repo path is a stage, not a destination.

### Route Treasure Chest access through a bare connector, no new role

Rejected for this stage. A same-module directory accessed through a plain
`RuntimeConnector` call would be an in-process function call dressed as a
connector, enforcing no real boundary. Jewelcrafter gives the isolation a
real, auditable seam before any module or repository boundary exists.

### Build the external-repository infrastructure now, in the same mission

Rejected — out of scope by explicit user direction. ADR-0032's T1–T7 remain
the correct home for that work, unchanged by this ADR.

## Consequences

### Positive

- Treasure Chest's code and its currently-hidden `internal/domain` coupling
  get an explicit, named boundary (Jewelcrafter) before any repository split,
  reducing the risk of ADR-0032's eventual external migration discovering
  new coupling at that point instead of now.
- The permanent migration guard (ADR-0012) gets an explicit verification
  step tied to the move, rather than assuming "no data deletion" alone
  covers command reachability too.
- No new build/release/module machinery is required yet — the staging step
  is cheap relative to ADR-0032's own external-repository stage 1.

### Negative

- Adds a role category (non-pipeline native role) that did not exist before;
  `strategist check`'s role-readiness reporting and `skill.yaml`'s role
  vocabulary documentation should note Jewelcrafter is not slot-bound, to
  avoid confusion with the three pipeline roles.
- Because there is no separate Go module yet, the isolation proof relies on
  convention/review (nothing imports `treasure-chest/` except through
  Jewelcrafter) rather than a compiler-enforced boundary, until ADR-0032's
  own stage begins.

## Addendum (2026-09-15) — Jewelcrafter file structure correction and design answers

**Mission:** `20260915-jewelcrafter-role-creation`

This ADR's original Decision item 2 specified `roles/jewelcrafter.yaml` +
`internal_skills/jewelcrafter/SKILL.md`. That is corrected above:
`domain.RoleConfig.Validate()` (`internal/domain/types.go:98-112`) requires
a `slot` field restricted to exactly `discovery`/`refinement`/`execution`
(`internal/domain/slots.go:17-33`) — there is no slot value for a
non-pipeline role, so a `roles/jewelcrafter.yaml` styled like
`roles/ranger.yaml`/`roles/archivist.yaml`/`roles/sniper.yaml` cannot
validate. This repository already has the correct precedent:
`internal_skills/scout/{skill.yaml,SKILL.md}`, whose own `SKILL.md` states
"unlike Ranger, Archivist, and Sniper, Scout is not a configurable slot and
has no `roles/scout.yaml` role-directives layer." Jewelcrafter follows the
same shape — `internal_skills/jewelcrafter/{skill.yaml,SKILL.md}` only.

Three further design questions this ADR left implicit are now answered:

- **Invocation trigger:** automatic, via Strategist's existing
  bootstrap/preflight stale-scan flow (`01-bootstrap.md`), non-blocking —
  the same slot as the existing mandatory stale scan and Keen Senses radar.
  Automatic from the start, independent of whether the backing
  implementation is today's in-repo `treasure-chest/` or a future
  externalized skill.
- **Relationship to `strategist treasure-chest scan`/`index`:** isolate,
  then delegate. Jewelcrafter never reimplements or duplicates
  treasure-chest's own mining logic — once `treasure-chest/` (or its future
  external successor) exists, Jewelcrafter calls into it for the
  jewels/potions slice via a `treasure_chest_delegate` reference. For other
  document kinds (ADRs, runbooks, generic docs), Jewelcrafter's mining logic
  is independent.
- **`.strategist/.data-offline/` schema:** a control layer analogous to
  `.strategist/.compiled/`'s own staleness pattern — an `index.yaml`
  manifest (one entry per curated item: id, source kind, source path,
  content digest, mined-at timestamp, TTL expiry, status) plus an `items/`
  directory holding the curated excerpt/summary per entry. See
  `.analysis/refined/20260915-jewelcrafter-role-creation/design.md` D3-revised
  for the full schema and the concrete `skill.yaml` content.

Building the bootstrap hook, the mining logic, and wiring
`treasure_chest_delegate` remain future, separately authorized
implementation work, combined with this repository's own `treasure-chest/`
relocation mission (`.analysis/refined/20260915-treasure-chest-relocation-verification/`)
into one implementation mission.

## Implementation boundary

This ADR records the accepted staging architecture. It does not implement the
relocation, the Jewelcrafter skill files, the migration-guard verification, or
any test/build changes. Those remain separate, separately authorized
implementation work — see `.analysis/refined/20260915-treasure-chest-root-path-isolation/tasks.md`
task group 2 (combined, per the 2026-09-15 addendum above, with
`.analysis/refined/20260915-jewelcrafter-role-creation/tasks.md` task group 2
and `.analysis/refined/20260915-treasure-chest-relocation-verification/tasks.md`
task group 1).
