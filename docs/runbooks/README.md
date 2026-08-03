# Runbooks: canonical source and runtime lookup policy

## Canonical source

`docs/runbooks/` is the canonical, source-controlled, human-reviewed location for
accepted agent-facing runbooks. A runbook here has already been reviewed and
accepted — it is not a candidate or a draft.

Each runbook documents a symptom, its root cause, and a verified resolution
sequence, following the shape used across this directory: `Symptom`, `Root Cause`,
`Resolution Steps`, `Reference`.

Entries fall into two categories. **Diagnostic** runbooks — every entry here so far —
follow the `Symptom`/`Root Cause`/`Resolution Steps`/`Reference` shape above: something
is broken and you're fixing it. **Procedural** runbooks instead capture a decision
checklist for a recurring situation where nothing is broken (`Trigger`/`Steps`/`Decision
Point`/`Stop Conditions`/`Reference`) — see `verifying-implemented-demands.md` for the
first one.

## Typed sidecars (`.runbook.yaml`)

Some runbooks additionally carry a co-located `docs/runbooks/<slug>.runbook.yaml`
sidecar — a structured counterpart to the markdown, not a replacement for it.
The markdown stays canonical for humans and for
`internal/treasure.ScanRunbookDirectory`'s Potion-scan `when_to_use` extraction
(which only ever reads `*.md` files; a `.runbook.yaml` sidecar is invisible to
it by construction — nothing there needs to change when a sidecar is added).
The sidecar exists so agent code can consume `applies_when`, `decision_gates`,
and leveled `checks` as data instead of re-parsing prose, via
`internal/runbook.ParseSidecar`, the single authoritative parser for this
schema.

### Schema

```yaml
schema_version: "1"
runbook_id: <slug matching the .md filename stem>
runbook_type: analytical | operational   # analytical: investigation/decision
                                          # (Ranger/Archivist side). operational:
                                          # execute-and-verify (Sniper side).
source_doc: docs/runbooks/<slug>.md      # the canonical markdown this sidecar describes

applies_when:
  - <trigger signal, one per item — required, non-empty>

objective: <one-line goal>               # required

preconditions:
  - <what must be true/known before starting>

analysis:                                # analytical runbooks: investigation steps
  - <investigation step>

decision_gates:
  - id: <slug>
    statement: <what must hold to proceed>

verification:                            # operational runbooks: post-fix checks
  - <check to run after the operational steps>

checks:
  - id: <slug>
    level: mandatory | recommended | conditional | informational
    when: <condition — only meaningful, and only allowed, when level: conditional>

metadata:
  version: 1
  owner: <optional — unset until a real owner is assigned>
  reviewed_at: <date this sidecar was last checked against its .md>
```

`schema_version`, `runbook_id`, `runbook_type`, `source_doc`, `objective`, and
a non-empty `applies_when` are required; `internal/runbook.ParseSidecar`
rejects a sidecar missing any of them, an unrecognized `runbook_type`, an
unrecognized `checks[].level`, or a `checks[].when` set on a non-`conditional`
check.

### Authoring guidance

Derive `applies_when` from the same "Trigger"/"Symptom" section the Potion
scan already reads for `when_to_use`, so the two stay traceable to one
source instead of drifting apart. Derive `objective` from the doc's own
title/opening line. Classify each `checks[].level` from the doc's own
language: "must"/"required" → `mandatory`, "should"/"recommended" →
`recommended`, "if applicable"/"when X" → `conditional`, purely informative
notes → `informational`. Authoring a sidecar means reading the paired `.md`
in full — it is not a mechanical transform, and two sidecars for unrelated
runbooks should not read as interchangeable. If a runbook's steps don't map
cleanly to a level, don't force it — flag the file for review instead of
guessing.

## Runtime-optimized artifacts are derived, not authoritative

Any runtime-optimized runbook artifact (a compiled index, summary, or mirror built
for faster lookup by an installed runtime) is a **derived cache**, never a second
source of truth. A runtime artifact MUST carry provenance metadata:

```yaml
source_path: docs/runbooks/<slug>.md
source_hash: <content-hash>
generated_at: <timestamp>
generator: <tool-or-command>
freshness: fresh|stale|unknown
```

As of this writing, no runtime-optimized runbook artifact exists yet — the
`runbook_opportunity` routine (see
`.strategist/contracts/machine/runbook-opportunity.yaml`) only creates reviewable
candidates, never a runtime index. This policy is documented ahead of that surface
being built so a future implementation has a lookup order to follow instead of
inventing one under time pressure. See `SQ-001` in
`.analysis/done/2026-07-25-quick-draw-runbook-opportunity/proposal.md` for the
deferred Treasure Chest / compiled-index integration question.

## Lookup order

1. If a source checkout is available, read `docs/runbooks/<slug>.md` directly (use
   a runtime index only to *locate* the slug, never as the content of record).
2. If a source checkout is unavailable but an installed runtime carries an
   optimized runbook artifact with provenance metadata, read the runtime copy and
   surface its `source_hash`/`freshness` alongside the content.
3. If both exist and `source_hash` disagrees with the canonical file's current
   hash, report the runtime artifact as stale and prefer canonical source.
4. If neither exists, fall back to normal discovery and, if appropriate, let the
   `runbook_opportunity` routine propose a new candidate.

## Candidates vs accepted runbooks

A runbook candidate produced by the `runbook_opportunity` routine (see
`.strategist/contracts/machine/runbook-opportunity.yaml#phases.sniper_runbook_opportunity.runbook_candidate_action`)
is a request for a runbook, not an accepted runbook. It is only promoted into this
directory after explicit human review — the routine itself never writes directly to
`docs/runbooks/<slug>.md`.
