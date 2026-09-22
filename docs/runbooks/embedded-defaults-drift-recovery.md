# Runbook: Recovering from embedded runtime-default drift

This runbook covers the recovery procedure after editing a file under
`internal/embed/defaults/` and observing a stale runtime diagnostic. It does
not authorize changing source files, CI configuration, or an installed runtime
when the mismatch is unexplained.

## Trigger

Use this procedure when `strategist check` reports one of these diagnostics for
a file listed by `internal/domain/runtime_defaults.go` in
`NormativeRuntimeDefaultFiles()`:

- `runtime_stale_conflict`
- `runtime_stale_auto_repairable`
- `runtime_stale_unknown_manifest`
- `runtime_newer_than_binary` (see "Binary older than the runtime" below)

The usual trigger is editing an embedded default and invoking a previously
built `strategist` binary, whose embedded copy still contains the old content.

## Root Cause

Three runtime layers can diverge:

1. `internal/embed/defaults/` — the authoring source in the checkout;
2. `.strategist/` — the installed runtime tree;
3. the `strategist` binary selected through `PATH` — which embeds the defaults
   present when it was built.

`strategist check` compares the installed file, the binary's embedded file, and
the corresponding entry in `.strategist/.install-manifest.json`. Editing the
authoring source does not update an already-built binary or an installed
runtime tree.

## Steps

### 1. Confirm the binary and checkout

From the intended repository checkout, record the binary that will actually be
used:

```bash
command -v strategist
stat "$(command -v strategist)"
git status --short -- internal/embed/defaults
```

Compare the binary timestamp with the edit that triggered the diagnostic. If
the resolved binary belongs to another checkout, package manager, or protected
installation, stop before replacing it.

### 2. Build an explicit candidate binary

Build to a temporary path so the current installation is not changed while the
candidate is being validated:

```bash
tmp_bin="$(mktemp -t strategist-drift-XXXXXX)"
go build -ldflags="-s -w" -o "$tmp_bin" ./cmd/strategist
```

The repository's equivalent build target may be used when it produces a known
binary path. Keep the output path explicit for the remaining checks.

### 3. Validate the candidate from the intended runtime root

Run the candidate binary against the checkout's `.strategist/` runtime root:

```bash
(cd .strategist && "$tmp_bin" check)
```

If the candidate reports the expected result for the edited file, replace the
active binary only when its path is owned and safely writable by the operator.
Use the repository's normal installation mechanism where available; do not
blindly overwrite a system-managed binary.

### 4. Re-run the installed check

After a safe replacement or installation, rerun the check from the same
checkout:

```bash
(cd .strategist && strategist check)
```

The previously reported `runtime_stale_*` diagnostic should be absent. A clean
result here confirms that the active binary and installed runtime agree for the
checked defaults.

### 5. Compare residual drift without overwriting it

If a `runtime_stale_*` diagnostic remains, compare the authoring and installed
copies for the exact normative path:

```bash
diff -- "internal/embed/defaults/<path>" ".strategist/<path>"
```

Repeat for every path named by `NormativeRuntimeDefaultFiles()`, not just the
first reported file. Do not copy one side over the other until the difference
has been classified: the installed runtime may contain an intentional local
edit, or the binary may still be from an unexpected installation.

### Binary older than the runtime (`runtime_newer_than_binary`)

The install manifest keeps each normative file's earlier installed hashes. When
the running binary carries a default the runtime already moved past, `check`
reports `runtime_newer_than_binary`, and `install`/`upgrade` refuse to write
instead of silently rolling the runtime back. This happens when one session or
checkout updated the runtime and another still uses an older `strategist` from
`PATH` (the 2026-09-22 incident: `~/.local/bin/strategist`, built before the
change, reverted freshly shipped contracts on every reinstall).

Update the binary, not the runtime:

```bash
make install                  # rebuilds ~/.local/bin/strategist from this checkout
strategist version --build
(cd .strategist && strategist check)
```

After changing anything under `internal/embed/defaults/`, run `make install`
before the next `strategist install` or `upgrade`, so the binary on `PATH`
carries the same defaults as the source. Pass `--allow-downgrade` only for a
deliberate rollback to an older release.

## Decision Point

A `runtime_stale_conflict` that survives rebuilding and safely installing the
candidate binary means the authoring source and installed runtime still
disagree, or the check is using a different binary/runtime root than expected.
Investigate the resolved paths, manifest entry, and full normative file set
before changing either source. Treat an unexplained mismatch as a separate
runtime incident rather than assuming that the authoring source should win.

## Stop Conditions

- The `strategist` path resolves to a system-managed, shared, protected, or
  otherwise ambiguous binary.
- The candidate binary cannot be built or validated from the intended
  `.strategist/` runtime root.
- A stale diagnostic remains after rebuilding and safe installation.
- The remaining diagnostic names a different normative file from the one being
  investigated.
- The source/runtime diff indicates an intentional local edit whose ownership
  has not been confirmed.

## Reference

- `internal/domain/runtime_defaults.go` — `NormativeRuntimeDefaultFiles()`,
  `DecideRuntimeDefaultUpdate`, and stale-diagnostic formatting.
- `internal/install/runtime_defaults.go` — install-manifest and runtime
  comparison behavior.
- [ADR-0044](../adr/0044-preflight-startup-consumes-preflightresult.md) — the
  rollout practice that motivated this recovery procedure.
- `.analysis/pending/20260916-runbook-candidate-embedded-defaults-drift-recovery.md`
  — original verified candidate and evidence boundary.
