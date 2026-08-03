# Runbook: Local release validation reaches the published-release asset gate

## Symptom

Local CI/CD or release validation prints:

```text
Published 6 artifact(s).
TAG is required ...
make: *** [Makefile:215: check-release-assets] Error 2
```

or, after the local skip behavior was added:

```text
Published 6 artifact(s).
::notice::skipping published-release asset gate because TAG is not set; local snapshot artifacts were already validated
```

## Root Cause

Two different release gates are being treated as one local check.

`check-release-artifacts` validates local GoReleaser output from
`dist/artifacts.json`. It is the snapshot/build-side gate. If it prints
`Published 6 artifact(s).`, GoReleaser produced the expected local release
artifacts and `dist/SHA256SUMS` exists.

`check-release-assets` validates a GitHub Release after publishing. It reads
`dist/published.tsv`, then uses `gh release view <tag>` to confirm that every
published artifact, its `.bundle`, and `SHA256SUMS` are retrievable from the
remote release. That gate only has meaning with a concrete release tag.

CI reflects this split:

- branch/PR local release validation runs `make release-dry-run`;
- tag-triggered release completion runs `make check-release-assets TAG="${GITHUB_REF_NAME}"`.

## Resolution Steps

### 1. Pick the gate that matches the environment

| Situation | Command |
|---|---|
| Validate GoReleaser config and local snapshot artifacts | `make release-test` |
| Install pinned GoReleaser, then run the local snapshot gate | `make release-dry-run` |
| Verify an already-published GitHub Release | `make check-release-assets TAG=vX.Y.Z` |

Do not use `check-release-assets` as the local snapshot gate. It is a remote
release-completeness check.

### 2. Interpret `Published 6 artifact(s).` correctly

This line is success from `scripts/check-release-artifacts.sh`. It means the
local manifest-driven artifact check completed.

If the next line complains about `TAG`, the failure is not that GoReleaser failed
to build artifacts. The command sequence continued into the published-release
asset gate without telling it which GitHub Release to inspect.

### 3. For local validation, stop at the local gate

Run:

```bash
make release-test
```

or:

```bash
make release-dry-run
```

These are the commands intended for local CI/CD validation before a release is
published.

### 4. For a real release, pass the tag explicitly

After a tag-triggered release has published assets and uploaded signature
bundles, run:

```bash
make check-release-assets TAG=vX.Y.Z
```

In GitHub Actions this is supplied by the release workflow as:

```bash
make check-release-assets TAG="${GITHUB_REF_NAME}"
```

## Stop Conditions

- If `dist/artifacts.json` is missing before local artifact validation, follow
  `release-no-dist-artifacts-found.md`.
- If the tag-triggered release fails with duplicate remote assets, follow
  `release-asset-already-exists.md`.
- If a release breaks on a run where no repository configuration changed, or a
  tool deprecation warning appears in CI, follow `release-tool-version-drift.md`.
- If `check-release-assets TAG=vX.Y.Z` fails after a real release, inspect the
  GitHub Release assets. Do not treat that as a local snapshot validation
  failure.

## Reference

Release history source of truth: `CHANGELOG.md` is curated for unreleased changes
and the `1.0.0` baseline. For patch releases after `1.0.0`, GitHub Releases are authoritative
for notes and published assets.

Diagnosed 2026-07-30 after a local validation sequence printed
`Published 6 artifact(s).` and then failed because `check-release-assets` was run
without `TAG`. The refined Strategist package is
`.analysis/refined/20260730-local-ci-cd-release-gates-runbook/`.
