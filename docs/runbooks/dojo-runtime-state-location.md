# Dojo Runtime-State Location

## Compatibility owner

The configured `<base_path>/dojo` domain owns current Dojo state. Keep the
following relative paths addressable across upgrade, reinstall, and workspace
moves:

| Artifact | Lifecycle | Backup expectation |
| --- | --- | --- |
| `<scenario>/criteria.yaml` | durable user-authored input | workspace backup |
| `.last-run/<scenario>/emit.log` | replaceable diagnostic evidence | optional |
| `.last-run/<scenario>/result.json` | retained latest derived evidence | retain when inspection matters |
| `.last-run/<scenario>/lesson.md` | durable failure learning | workspace backup |
| `.history.jsonl` | durable aggregate learning history | workspace backup |

`.strategist` is generated runtime state and is not the owner of durable Dojo
learning. Replacing it must not remove or redirect the workspace Dojo paths.

## Judge calibration case state

A `judge_calibration` case (ADR-0066) adds artifacts to the same domain without moving any existing
path. Its storage contract is `dojo.CalibrationArtifactPolicies()`:

| Artifact | Location | Class | Backup |
| --- | --- | --- | --- |
| raw observations | `<case output>/observations.jsonl` | durable | workspace backup |
| latest run evidence (also for a passing run) | `<case output>/evidence.json` | retained | when inspection history matters |
| audit of generated negatives | `<case output>/audit.jsonl` | durable | workspace backup |
| corpus manifest (references and digests) | `<case output>/corpus.json` | replaceable | optional |
| published calibration record | `<base_path>/dojo/calibrations/<id>.json` | durable | workspace backup |
| golden promotions and candidates | `<base_path>/dojo/corpus/*.jsonl` | durable | workspace backup |
| operator proposals | `<base_path>/dojo/mutations/proposals.jsonl` | durable | workspace backup |

`corpus`, `mutations` and `calibrations` are reserved names: no scenario or case may use them and
`dojo list` never reports them. `<case output>` defaults to `<base_path>/dojo/<case>/` and may be set per
case with `calibration.output`; the path must stay inside the workspace and outside `refined`, `done`,
`pending`, `todo`, `archived` and `.strategist`. Changing the output of an existing case is a path
change: it needs the preview, preservation, recovery and rollback listed under *Future migration
boundary* before any state is moved. A calibration record is never overwritten (its id is derived from
its content), and the handoff reads it only through the id pinned in `.strategist/integrations.yaml`.

## Persistence and recovery

Latest results are replaced atomically. Dojo state writes are serialized so
concurrent checks cannot interleave JSONL records or overwrite a result with a
partial document. A checker verdict is authoritative independently of a
subsequent persistence warning; retry the persistence operation when the
warning identifies a storage failure.

`InspectStorage` provides a read-only recovery view for malformed results,
malformed history lines, and state whose scenario criteria are absent. Do not
delete or rewrite the source based only on that report.

## Future migration boundary

There is no automatic first-run move, cleanup, symlink, or dual write. A future
path change must be accepted separately and must provide:

1. a read-only inventory and preview of existing state;
2. source preservation until verification succeeds;
3. visible partial-failure and resumable recovery status; and
4. rollback semantics before write paths change.

The incidental Sniper session-collision workflow is not part of this Dojo
state contract.
