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
