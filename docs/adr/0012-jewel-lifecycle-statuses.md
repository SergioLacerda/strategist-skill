# ADR-0012 — Jewel lifecycle statuses supersede the active/deprecated model

**Status:** Accepted
**Date:** 2026-07-14
**Context:** `treasure-chest-index-mine-pipeline`, supersedes the status portion of ADR-0011

---

## Context

ADR-0011 established that jewels are generated and activated immediately by the analyzing
agent (`reviewed_by: agent`, `status: active` on creation), with a trust-tier ceiling as the
only safeguard — no human pre-approval gate.

In practice this made `active` do two jobs at once: "this jewel exists and is not
deprecated" and "this jewel is trustworthy enough to use as runtime evidence." Those are
different claims. An agent-generated jewel with no human review should be usable as a
ranking/token-economy hint, but must never be indistinguishable from a jewel a human has
actually reviewed and confirmed against real evidence.

The `treasure-chest-index-mine-pipeline` mission also introduces `strategist treasure-chest
index`, which generates candidate jewels in bulk from mission-history clustering — at a
volume where "immediately active" is no longer an acceptable default; bulk-generated
candidates need an explicit lower-trust holding state distinct from anything a human has
touched.

## Decision

Replace the two-state `active | deprecated` status with a four-state lifecycle:

- `proposed` — agent- or `index`-generated, not yet human-curated. Usable only as a
  ranking/token-economy hint during retrieval, never as verified evidence.
- `accepted` — promoted by `strategist treasure-chest mine --accept`. A human has reviewed
  the statement and judged it sound.
- `verified` — promoted by `strategist treasure-chest mine --verify --evidence <ref>`. A
  human has recorded concrete validation evidence, not just reviewed the statement.
- `deprecated` — no longer usable at runtime. Reached manually (`mine --deprecate`) or
  automatically when the parent chest is tombstoned via `treasure-chest remove`. Terminal:
  a deprecated jewel can never be promoted back.

`active` is removed, not aliased or silently accepted. `ValidateJewelStatus`
(`internal/domain/jewel_grade.go`) rejects it explicitly with a migration hint, and
`loadJewels` (`cmd/strategist/treasure_chest_jewels.go`) fails loudly rather than degrading
silently when it encounters a legacy `active` entry. A one-time, idempotent migration path —
`strategist treasure-chest mine --migrate-status` — rewrites every `status: active` entry to
`status: accepted` (the closest equivalent: a jewel an agent had already put into runtime use
under the old model, now made explicit as human-owned going forward since no verification
evidence exists for it).

Runtime retrieval (`jewel_retrieval` in
`internal/embed/defaults/contracts/machine/context-enrichment.yaml`) now has an explicit
status precedence: `verified` preferred first, `accepted` preferred next, `proposed` allowed
only as a hint, `deprecated` excluded entirely. This does not change ADR-0011's core
trade-off — there is still no human pre-approval gate blocking jewel *generation* — it only
sharpens what a jewel's status is allowed to claim about itself before a human has looked at
it.

This decision does not revisit the trust-tier ceiling (`ValidateJewelTrust`), which is
unaffected and continues to apply to every status.

## Consequences

**Positive:**
- Retrieval can distinguish "an agent thinks this might be useful" from "a human confirmed
  this is true," without reintroducing a pre-approval bottleneck on generation.
- Bulk `index`-generated candidates have a natural home (`proposed`) that cannot be mistaken
  for curated knowledge, even before any human looks at the curation queue.
- The legacy status is rejected loudly instead of silently tolerated, so drift from an
  unmigrated `jewels.yaml` surfaces immediately instead of degrading retrieval quality
  invisibly.

**Negative:**
- Existing installs must run `mine --migrate-status` once; skipping it means `loadJewels`
  (and therefore `strategist treasure-chest`) errors on any chest with legacy jewels until
  migrated.
- Four states are more to reason about than two. The `mine` command's non-interactive flags
  (`--accept`/`--verify`/`--deprecate`/`--migrate-status`) exist specifically to keep that
  complexity scriptable rather than requiring a human to hand-edit YAML.

## Reference

- Originating mission: `treasure-chest-index-mine-pipeline`
- Supersedes (status portion only): [ADR-0011](0011-jewel-promotion-trust-ceiling-exception.md)
  — the trust-tier ceiling itself is unaffected
- Enforcement: `internal/domain/jewel_grade.go` (`ValidateJewelStatus`)
- Runtime: `cmd/strategist/treasure_chest_index.go`, `cmd/strategist/treasure_chest_mine.go`
- Contract: `internal/embed/defaults/contracts/machine/context-enrichment.yaml`
  (`jewel_generation`, `jewel_retrieval`)
- Docs: `cli-reference.md` § Jewels
