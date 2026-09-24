# ADR-0052 — CI/CD Enforcement Policy

**Status:** Proposed
**Date:** 2026-09-24
**Related:** ADR-0014, ADR-0023, ADR-0051

## Context

The v1.0.22 release chain is strong: deny-by-default workflow permissions,
SHA-pinned actions (one exception, `actions/upload-artifact@v4`), re-verification
of the tagged commit, GoReleaser, keyless Cosign signing, a CycloneDX SBOM,
build-provenance attestation and a post-publish asset check. A review of the
repository found the gap between *running* CI and *enforcing* it:

- The Health Orchestrator jobs and CodeQL run on every PR, but whether they are
  required to merge is a GitHub ruleset setting that the repository cannot prove.
- The release workflow re-runs the tagged commit's gates but does not check that
  the commit is reachable from `main`, that the tag is annotated, or that the tag
  version matches the project version. Existing tags (`v1.0.20`–`v1.0.22`) are
  lightweight, and reachable from `main` only by practice.
- Post-publish verification checks that assets, bundles and `SHA256SUMS` exist;
  it does not verify signatures or attestations.
- The project has one maintainer (`GOVERNANCE.md`), so policies that require a
  second person must keep an explicit maintainer path.

The race detector already gates Linux CI (`go test -race ./...`), so no new gate
is added for it.

## Decision

1. **Required checks.** `main` requires the seven Health Orchestrator jobs
   (`lint`, `test-windows`, `test`, `security`, `validate`, `site-build`,
   `release-dry-run`) and both CodeQL legs. The check names are confirmed from a
   real PR before the ruleset is changed.
2. **One required review, with a maintainer bypass.** The `main` ruleset requires
   one approving review (applied by the maintainer), dismisses stale approvals and
   requires conversation resolution. A bypass actor limited to pull requests keeps
   the sole maintainer able to merge. Code Owners review and last-push approval are
   not enabled while `CODEOWNERS` lists only the maintainer. The bypass is dropped
   when a second active maintainer exists.
3. **Tag governance.** A `v*.*.*` tag ruleset restricts creation, update and
   deletion, with a documented admin bypass. Release tags must be annotated. A
   release-tag check in the `verify` job fails unless the tagged commit is an
   ancestor of `origin/main` and the tag is annotated and consistent with the
   project version.
4. **Signed tags are deferred.** GitHub tag rulesets cannot require signatures and
   verifying them in CI needs key distribution; the decision is revisited with the
   `release` Environment.
5. **Windows promotion by data.** The full Windows suite stays non-blocking until
   30 consecutive green runs, zero unexplained failures and an in-budget duration
   are recorded. Every `continue-on-error` carries an owner, reason and removal
   condition, checked by a script.
6. **Workflow static analysis stays out of the required set until stable.**
   `actionlint`, `zizmor` and `shellcheck` run with pinned versions and become
   required only after a clean baseline.
7. **Verification depth.** The release dry-run gains `strategist version` on the
   built binary, checksum coverage and SBOM correlation; the release workflow
   gains post-publish `cosign verify-blob` and `gh attestation verify` steps.

## Alternatives considered

- **Zero required approvals** (the initial proposal): superseded by the
  maintainer's decision; it remains the documented fallback if the bypass proves
  unworkable.
- **Ruleset-only tag protection:** rejected as the sole control; it is not
  provable or testable from the repository, so the release-tag check complements it.
- **Keep lightweight tags:** rejected; they carry no tagger, date or message.
- **Promote the Windows suite immediately:** rejected without flakiness data.
- **Add a Linux race gate:** unnecessary, it already exists.

## Consequences

- Merging to `main` and publishing a release depend on settings that live on
  GitHub. `docs/runbooks/cicd-enforcement-settings.md` records the target state, the
  read-only queries that verify it and the rollback.
- Wrong required-check names can block every merge; the runbook makes name
  confirmation a precondition.
- The first annotated tag is the cutover from lightweight tags.
- Implementation of the repository and settings changes is tracked outside this
  ADR in `.analysis/refined/20260924-cicd-hardening-v1-0-22-refinement/tasks.md`.
  Until it lands, this ADR describes intended policy, which is why it is
  `Proposed`.
- Open: the "branch must be up to date" flag on `main`, decided after observing a
  `develop` → `main` PR cycle.
