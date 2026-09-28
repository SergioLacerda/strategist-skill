# Release Checklist — v1.0.25

**Status:** Mission-specific checklist for one release (steps 1/3/5 not yet executed)
**Last Updated:** 2026-09-28

This is a mission-specific checklist for shipping this one release; the
general, authoritative procedure lives in `CONTRIBUTING.md` ("Cutting a
release tag") and `docs/runbooks/local-ci-cd-release-gates.md` — this
document sequences them for `v1.0.25` and does not replace either.

`v1.0.25` is the version-scheme-correct next tag: this repo uses a flat
sequential patch counter (`v1.0.3` … `v1.0.24`), not conventional-commits
semver.

## Current state (checked 2026-09-28)

- `origin/develop` and `origin/main` are already in sync content-wise:
  `develop`→`main` landed via PR #71, and `main`'s tip is a merge commit for
  that PR one ahead of local `develop`'s own last `main`-merge commit — no
  new file changes, just the merge itself. **Step 1 below is a
  confirmation, not a pending PR to land**, unlike the `v1.0.23` checklist.
- 18 commits have landed on `develop` since `v1.0.24` was tagged: a run of
  `develop`→`main` PR merges (#59–#63, #66–#67), three Dependabot dependency
  bumps merged directly (`#64` npm/web-landing, `#65` GitHub Actions,
  `#68` Go `spf13/pflag`, `#69` npm/web-landing, `#70` GitHub Actions), and
  the CI/bug fixes from this repository's own hardening pass earlier this
  week (Windows `filelock` contention retry, `strategist check`'s
  ranked-runtime executable gate no longer misfiring for `host`/`embedded`
  runtimes, two `tests/spec` parity tests no longer depending on a
  self-installed `.strategist/` at the repo root, and the docs-governance
  fixes for `docs/release-checklist-v1-0-23.md`).
- `v1.0.24` is an annotated tag; the annotated-tag rule from `v1.0.23`
  onward is now business as usual, not a special case to call out.

## Steps

1. **Confirm `develop`↔`main` are in sync.**
   ```bash
   git fetch origin main develop
   git log --oneline origin/develop..origin/main   # expect: only the PR #71 merge commit itself
   git log --oneline origin/main..origin/develop   # expect: empty
   ```
   If either check shows unexpected commits, land or investigate before
   continuing — do not assume the state above still holds by the time you
   run this.
   **`[manual — human executes; read-only, but gates the rest of this checklist]`**

2. **Run the local tag-check gate.**
   ```bash
   git switch main && git pull --ff-only
   make release-tag-test                      # the check itself, against throwaway repos
   ```
   Safe to run locally; does not push or mutate remote state.

3. **Cut the annotated tag.**
   ```bash
   git tag -a v1.0.25 -m "v1.0.25"              # annotated, never `git tag v1.0.25`
   ```
   **`[manual — human executes; git mutation, out of Strategist/Sniper scope]`**

4. **Verify the tag against the same check the workflow runs.**
   ```bash
   make release-tag-check TAG=v1.0.25 MAIN_REF=main   # same check the workflow runs
   ```
   Safe to run locally; does not push or mutate remote state.

5. **Push the tag to trigger the release.**
   ```bash
   git push origin v1.0.25
   ```
   **`[manual — human executes; git mutation, out of Strategist/Sniper scope]`**

6. **After CI publishes, verify the published release.**
   ```bash
   make check-release-assets TAG=v1.0.25
   ```
   Do not run this before the tag is pushed and the Release workflow has
   completed — it checks the *published* GitHub Release
   (`dist/published.tsv` + `gh release view`), not the local build; see
   `docs/runbooks/local-ci-cd-release-gates.md` if it fails.

## Optional: CHANGELOG `[Unreleased]` hygiene

Before tagging, optionally review `CHANGELOG.md`'s `[Unreleased]` section
for bullets that already shipped in an earlier tag, using
`docs/runbooks/changelog-shipped-verification.md`'s procedure (`git log
-S`, `git tag --contains`, boundary-size cross-check). That runbook is
marked "first draft, single verified use" — treat it as optional guidance,
not a hard gate. `CHANGELOG.md` itself is not backfilled with `v1.0.x`
release notes; GitHub Releases are authoritative for those
(`CONTRIBUTING.md` § Release history).

## Out of scope for this checklist's authoring

Steps 3 and 5 above (the tag creation and the tag push) are git-mutating
actions. They are named here for completeness but were not — and will not
be — executed by whoever authored this document: git mutation is outside
the default Sniper execution contract and outside this project's own
standing execution rules. A human maintainer runs those two steps directly.
Step 1 is read-only but still requires a human to interpret and confirm.
