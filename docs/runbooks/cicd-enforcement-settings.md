# Runbook: CI/CD enforcement settings (GitHub)

## Trigger

You are configuring, auditing or changing the GitHub-side enforcement of this
repository's CI/CD: the branch ruleset on `main`, the tag ruleset for release
tags, the `release` Environment or the repository security features. Also use it
when a merge or a release is unexpectedly blocked (or unexpectedly allowed).

These settings live on GitHub, not in the repository, so nothing in the checkout
can prove them. Every step below therefore starts from a read-only query; capture
its output before changing anything.

## Steps

1. **Capture the current state (read-only).** Attach the JSON to the change that
   modifies the settings.

   ```bash
   gh api repos/SergioLacerda/strategist-skill/rulesets
   gh api repos/SergioLacerda/strategist-skill/rulesets/<id>
   gh api repos/SergioLacerda/strategist-skill/rules/branches/main
   gh api repos/SergioLacerda/strategist-skill --jq '.security_and_analysis'
   gh api repos/SergioLacerda/strategist-skill/environments
   gh run list --workflow "Health Orchestrator" --limit 50
   ```

2. **Confirm the required-check context names.** Read them from a recent PR, not
   from the workflow files: the name GitHub matches is the reported check name.

   ```bash
   gh pr checks <pr-number> --json name,workflow,state
   ```

   Health Orchestrator jobs: `lint`, `test-windows`, `test`, `security`,
   `validate`, `site-build`, `release-dry-run`. CodeQL is a separate workflow
   whose matrix legs report as `Analyze (go)` and
   `Analyze (javascript-typescript)`. Use the names the command prints.

3. **Branch ruleset for `main`** (target state):
   - require a pull request before merging;
   - require status checks: the seven Health Orchestrator jobs and both CodeQL
     legs, using the names from step 2;
   - require one approving review, dismiss stale approvals on new commits, and
     require conversation resolution;
   - keep block deletion and block non-fast-forward;
   - keep a bypass actor (repository admin/maintainer) limited to **pull
     requests only**. A single maintainer cannot approve their own PR, so without
     this every merge blocks;
   - do **not** enable "require review from Code Owners" or "require approval of
     the most recent reviewable push" while `.github/CODEOWNERS` lists only the
     maintainer: neither can be satisfied.

4. **The "branch must be up to date" (strict) flag.** Leave it off until at least
   one `develop` → `main` PR cycle has been observed; a long-lived `develop`
   branch turns strict mode into constant refresh work. Record the decision and
   its date in the change that sets it.

5. **Tag ruleset for `v*.*.*`:** restrict creation, update and deletion, with a
   documented admin bypass. Release tags are annotated (`git tag -a vX.Y.Z -m
   "..."`); lightweight tags are not accepted once the release-tag check is in
   place. Tag signing is recommended but not enforced (GitHub tag rulesets cannot
   require signatures); it is re-evaluated with the `release` Environment.

6. **`release` Environment:** limit which actors can publish, optionally add a
   wait timer, and keep the release secrets and permissions on that Environment.
   Manual approval is optional while there is one maintainer.

7. **Security features:** enable secret scanning and push protection, then
   confirm with the `security_and_analysis` query from step 1.

8. **Verify the effect** with a throwaway PR: a failing or pending required check
   must block merge; a self-authored PR with green checks must be mergeable
   through the bypass path.

## Decision Point

- **Required review keeps blocking the maintainer's own PRs:** the bypass actor is
  missing or not limited correctly. Fix the bypass; do not disable required
  checks to unblock a merge.
- **A required check is never reported:** the context name is wrong (step 2).
  Correct the name, do not remove the requirement.
- **Second active maintainer:** drop the maintainer bypass for pull requests and
  reconsider Code Owners review and last-push approval.

## Rollback

Each setting is reversible from the ruleset UI or API: set the ruleset's
enforcement to `evaluate` (or `disabled`) to observe without blocking, then
restore it. Keep the captured JSON from step 1 as the restore point. An emergency
release uses the documented admin bypass; record why in the release notes.

## Stop Conditions

- The current settings cannot be read (no `gh`, no access): do not guess; ask the
  maintainer for the API output.
- The required-check names cannot be confirmed from a real PR.
- A change would leave `main` or the release tags with no bypass path for the
  only maintainer.

## Reference

- `.github/workflows/test.yml`, `.github/workflows/release.yml`,
  `.github/workflows/codeql.yml`
- `.github/CODEOWNERS`, `GOVERNANCE.md`, `MAINTAINERS.md`, `SECURITY.md`
- `.analysis/refined/20260924-cicd-hardening-v1-0-22-refinement/design.md`
  (DEC-002, DEC-003, DEC-005)
