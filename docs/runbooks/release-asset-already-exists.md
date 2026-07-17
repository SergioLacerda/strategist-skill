# Runbook: Release upload fails with `422 already_exists`

## Symptom

The `Release` GitHub Actions workflow (`.github/workflows/release.yml`) fails during
the `goreleaser release --clean` step, after build and archiving succeed, with
errors like:

```
upload failed  error=POST .../releases/<id>/assets?name=strategist-<os>-<arch>:
  422 Validation Failed [{Resource:ReleaseAsset Field:name Code:already_exists}]
⨯ release failed after ...  error=scm releases: failed to publish artifacts: ...
```

## Root Cause

A GitHub release for the pushed tag already exists with one or more of the same
asset names attached — typically because the workflow was rerun (or the tag was
re-pushed) after an earlier attempt had already uploaded some/all assets.
`goreleaser release --clean` only clears the local `dist/` directory; it does not
touch assets already published on the remote GitHub release. Without
`release.replace_existing_artifacts: true` in `.goreleaser.yaml`, GoReleaser
attempts to create the asset by the same name again, and GitHub's API rejects the
duplicate with `422 already_exists`.

## Fix Applied

`.goreleaser.yaml` now sets `release.replace_existing_artifacts: true`, so a rerun
against a tag whose release already has assets overwrites them instead of failing.

## What To Do If It Still Fails

1. Check whether the release for the tag already exists and inspect its assets:
   `GET https://api.github.com/repos/<owner>/<repo>/releases/tags/<tag>`
2. If assets are missing or corrupted and the fix above isn't sufficient, delete the
   conflicting release/assets on GitHub before rerunning the workflow, or cut a new
   patch tag instead of rerunning against the same tag.
3. Do not rerun the `Release` workflow repeatedly against the same tag as a first
   response — prefer a new patch tag if the fix doesn't resolve it immediately.

## Reference

Diagnosed 2026-07-16 for the `v1.0.6` release (release-id `355261405`): the release
was created at `17:08:23Z`; assets were (re-)uploaded around `17:37:0xZ`, ~29
minutes later — consistent with a workflow rerun against the same tag/release.
Analysis: `.analysis/refined/20260716-bug-ci-cd-release/`.
