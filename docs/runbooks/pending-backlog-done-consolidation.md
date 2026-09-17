# Runbook: Pending-Backlog Done-Consolidation

## Trigger

A backlog directory under `<base_path>/pending/` (or a subsystem folder
within it) has accumulated multiple mission packages, some of which have
already reached `mission_status: documentation_applied`, with no separation
from still-open work.

## Steps

1. **Inventory.** List every package/file directly under the backlog root;
   read each package's own `analysis.md`/`tasks.md` frontmatter for
   `mission_status` — do not infer status from folder name, checkbox state,
   or a Sniper report existing elsewhere (same evidence bar as
   `11-critical-hit.md`'s Insufficient Evidence list).
2. **Partition** into: done (`mission_status: documentation_applied`, or a
   flat pending demand a done package's own `tasks.md` documents as
   resolved), still-open (any other status), and seed/source material
   (pre-existing drafts the backlog was refined from, never a
   Strategist-mission output themselves).
3. **Consolidate done work.** Create `<backlog_root>/done/`; move each done
   package into it unchanged. For a flat resolved demand, move it in with a
   short prepended resolution note (date + which package resolved it) —
   never delete, never rewrite the original content below the note.
4. **Write the root index.** `<backlog_root>/README.md`: one line/section
   per group from step 2 — done items linking into `done/`, still-open
   items named as open (not omitted), seed material named as such with a
   pointer to where it lives.
5. **Promote the roadmap, if one exists.** If a roadmap/plan document
   already exists as seed material, move it to `<backlog_root>/ROADMAP.md`
   (or equivalent root name) with a dated status block prepended above the
   original content — leave the original wave/item descriptions untouched
   below the block. Update the one seed-material index that pointed at its
   old path.
6. **Consolidate what's left.** Write exactly one new, unrefined pending
   item at `<backlog_root>/<date>-<topic>-remaining-work.md` consolidating
   every not-started/partial item surfaced in steps 2 and 5 into a single
   flat reference — not a refined mission package.
7. **Fix live cross-references only.** Fix cross-references only in
   still-open, actively-maintained packages that point at a path moved in
   step 3 or 5. Do not rewrite historical (done, ADR, archived-report)
   packages' own internal paths — add one path-resolution note to the
   step-4 README instead.
8. **Leave the reorganization mission itself in place.** Leave this
   reorganization's own package where it is (`pending/`, not `done/`) — a
   later pass over the same backlog is what will eventually move it, the
   same way it just moved everything else.

## Decision Point

- **mission-status-evidence-sufficient** — A package's own frontmatter
  `mission_status` field (not a Sniper report's existence, not a
  fully-checked `tasks.md`, not the Approval Gate having been accepted) is
  what Step 2 uses to classify it as done vs. open — evidence bar matches
  `11-critical-hit.md`'s Insufficient Evidence list.
- **seed-material-vs-mission-output** — A document is seed/source material
  (Step 2's third bucket) only if it predates and was never itself a
  Strategist-refined mission package (`analysis.md`/`proposal.md`/
  `design.md`/`tasks.md`) — an already-refined package that happens to
  still be open is still-open, not seed material.

## Stop Conditions

- A package's status can't be confidently read from its own frontmatter
  (ambiguous or missing `mission_status`) — stop and ask, do not guess from
  surrounding context.
- The backlog contains layered sub-taxonomies, thematic domains, or is
  large enough to need a scripted count-invariant check — this runbook's
  lighter procedure does not fit; use
  `docs/runbooks/demands-and-docs-reorganization.md` instead.
- Any move would place a file outside the backlog's own `base_path`, or
  would touch non-`.md` files — outside this runbook's scope; fall back to
  `main_mission` judgment.

## Reference

- `docs/runbooks/demands-and-docs-reorganization.md` — heavier sibling
  procedure for larger, differently-shaped backlogs (layered taxonomies /
  thematic domains / scripted migration).
- `.strategist/contracts/narrative/11-critical-hit.md` — Insufficient
  Evidence list this runbook's status-classification step (Decision Point 1)
  is bound by.
- Worked example: `.analysis/pending/cli_refactor/README.md`,
  `.analysis/pending/cli_refactor/ROADMAP.md`, and
  `.analysis/pending/cli_refactor/20260916-cli-refactor-remaining-work.md`
  — the concrete output of following this procedure once.
