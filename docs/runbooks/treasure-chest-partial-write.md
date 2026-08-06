# Runbook: Partial `treasure-chest add`/`remove` Write

## Symptom

`strategist treasure-chest add` or `strategist treasure-chest remove` exits
non-zero partway through, and it's unclear whether `active.yaml`,
`treasure-chests.yaml`, `knowledge.index.yaml`, or a cascaded jewel file (on
`remove`) were left in an inconsistent state.

Two distinct variants, distinguishable by the error text:

**Variant A — prepare-phase failure** (no `(already committed: ...)` in the
error):

```
write <path>: create temp: ...
write <path>: encode: ...
```

**Variant B — commit-phase failure** (error contains `(already committed:
...)`):

```
write <path> (already committed: [<path1> <path2> ...]): rename temp: ...
```

## Root Cause

`treasure-chest add`/`remove` write `active.yaml`, `treasure-chests.yaml`,
`knowledge.index.yaml`, and (on `remove`) cascaded jewel files as a single
all-or-nothing batch (`internal/treasure/yaml_node.go` → `WriteYAMLNodes`):

1. **Prepare phase** — every document is serialized and written to a temporary
   sibling file next to its destination. If any of these writes fails, every
   temp file staged so far in the batch is cleaned up and **no destination
   file is touched** — this is Variant A. The workspace is exactly as it was
   before the command ran.
2. **Commit phase** — only entered once every prepare step succeeds. Each temp
   file is renamed into place, in order. If a rename fails partway (rare — the
   OS revoking permissions mid-run, or a destination path unexpectedly
   becoming a directory), the files renamed before the failure now hold new
   content, and the remaining files (including the one that failed) still
   hold old content — this is Variant B, and the error names exactly which
   paths are on which side.

## Resolution Steps

1. Read the error text. If it does **not** contain `(already committed:
   ...)`, this is Variant A: nothing was mutated. Fix the underlying cause
   (free disk space, fix permissions, create a missing parent directory) and
   simply re-run the same `add`/`remove` command.
2. If the error **does** contain `(already committed: ...)`, this is
   Variant B: some files already hold new content, others still hold old
   content. Run:
   ```
   strategist treasure-chest doctor
   ```
   to get an exact per-chest report of which of `active.yaml`,
   `treasure-chests.yaml`, and `knowledge.index.yaml` the affected chest is
   present in and absent from.
3. `doctor` only detects — it does not repair. Using its report, manually
   bring the files back into agreement: either re-run `add`/`remove` for the
   affected chest id (safe to retry — the command overwrites/tombstones
   deterministically by id), or hand-edit the lagging file to match the
   others.
4. Re-run `strategist treasure-chest doctor` and confirm it reports no
   consistency drift before continuing.

## Reference

- `internal/treasure/yaml_node.go` → `WriteYAMLNodes`, `writeTempSibling`
- `internal/treasurecli/treasure_chest_doctor.go`
- `docs/cli-reference.md` § `treasure-chest add` / `treasure-chest remove`, §
  `treasure-chest doctor`
- `.analysis/done/2026-07-22-treasure-mutation-transactionality/completion-report.md`
