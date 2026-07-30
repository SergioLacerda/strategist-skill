# Runbook: Release fails with `No dist/strategist-* artifacts found`

## Symptom

The `Release` GitHub Actions workflow (`.github/workflows/release.yml`) fails
*after* `goreleaser release --clean` reported success. GoReleaser's own log shows
it building, archiving and uploading `strategist-<os>-<arch>` assets to the
GitHub release, yet the very next step fails with:

```
Run shopt -s nullglob
Error: No dist/strategist-* artifacts found after goreleaser run
Error: Process completed with exit code 1
```

The release itself is created and its assets *are* present on GitHub — only the
post-processing steps (cosign signing, bundle upload, provenance attestation)
never run.

## Root Cause

The post-processing steps globbed `dist/strategist-*`, assuming the archive
`name_template` determines the filename on disk. It does not.

`archives[].name_template` names the **published asset**. With
`archives[].formats: [binary]` GoReleaser does not create a new file at all — it
registers the already-built binary as the uploadable artifact and renames it only
at upload time. With the previous build config (`binary: strategist`, default
`no_unique_dist_dir: false`), the binaries lived at:

```
dist/strategist_linux_amd64_v1/strategist
dist/strategist_darwin_arm64_v8.0/strategist
...
```

so `dist/strategist-*` (hyphen, at dist root) matched **zero** files. `nullglob`
made the loops silently no-op, which is precisely why the failure surfaced as a
confusing "nothing was built" error rather than a layout mismatch.

## Fix Applied

The durable fix is manifest-driven artifact selection.

`.goreleaser.yaml` intentionally keeps the plain build binary name
(`binary: strategist`) and GoReleaser's default per-target `dist/` layout. A
previous attempt to make on-disk filenames match published asset names by using a
target-templated binary name plus `no_unique_dist_dir: true` caused duplicate
artifact-name collisions when `formats: [binary]` re-registered the same file at
the archive stage.

The release workflow no longer depends on either layout. Its `Collect published
artifacts` step reads `dist/artifacts.json` (GoReleaser's machine-readable
manifest), selects published `Binary` / `Archive` entries, verifies each listed
path exists and is non-empty, and feeds those exact paths to the signing and
attestation steps.

Signature bundles are now written to `dist/bundles/<asset-name>.bundle`, keeping
the published pair `<asset>` + `<asset>.bundle` documented in `README.md` and
`SECURITY.md`, and keeping bundles out of the attestation subject set.

## What To Do If It Still Fails

1. Read the step output of `Collect published artifacts` — on failure it dumps
   every `type`/`name` pair found in `dist/artifacts.json`. If there are no
   `Binary` or `Archive` entries, the problem is in `.goreleaser.yaml` (build or
   archive config), not in the workflow.
2. Never "fix" this by widening a glob. If a path assumption is needed, derive it
   from `dist/artifacts.json`; GoReleaser's on-disk layout is an implementation
   detail that changes with build options and with GoReleaser versions.
3. Reproduce locally without publishing anything:
   `goreleaser release --snapshot --clean && jq -r '.[] | [.type,.name,.path] | @tsv' dist/artifacts.json`
4. If the release was already created before the failure, follow
   `release-asset-already-exists.md` before rerunning against the same tag.
5. Before cutting another tag, run the local release test:
   `make release-test`
6. In CI, the branch/PR dry-run gate is `make release-dry-run`; it installs the
   pinned GoReleaser version and then runs the same manifest-driven release test.

## Reference

Diagnosed 2026-07-29. Local `dist/` from a prior run showed the per-target
directory layout (`dist/strategist_<goos>_<goarch>_<variant>/`) with no
hyphenated file at `dist/` root, confirming the glob could never have matched.
