# Runbook: CHANGELOG "Already Shipped?" Verification

## Trigger

You are preparing `CHANGELOG.md`'s `[Unreleased]` section and need to
confirm whether a specific bullet describes work that has already shipped
in a prior tagged release, rather than reading full project history from
scratch to answer that question.

## Known Limitation

GitHub Release bodies (goreleaser's raw commit-list format) do not map 1:1
to `CHANGELOG.md`'s curated prose bullets. Semantic matching (Step 1 below)
is still required — this procedure narrows *where* to look, it does not
fully automate the match.

## Steps

1. Identify a candidate commit that introduced the feature/change described
   by the bullet: `git log -S"<distinctive phrase or code token>" --oneline
   -- <path>`, or `git log --diff-filter=A --oneline -- <path>` if the
   bullet describes a whole new file/feature.
2. `git tag --contains <commit-sha>`, sorted by version — the first
   (oldest) tag listed is the earliest release containing that commit.
3. Cross-check the release-boundary size: `git log <tag_n>..<tag_n+1>
   --oneline | wc -l` — a tag range with very few, non-`feat:` commits
   strongly implies nothing substantial shipped in that gap, corroborating
   that a bullet must have shipped earlier.
4. Optional corroboration: fetch the published GitHub Release body via the
   public API (`curl -s
   https://api.github.com/repos/<owner>/<repo>/releases/tags/<tag>`, no
   auth needed for a public repo) to confirm goreleaser's auto-generated
   commit-list notes independently agree with the `git tag --contains`
   result.

## Decision Point

**All checks corroborate an earlier release** (candidate commit identified,
its earliest containing tag found, boundary size small/non-`feat:`, and —
if checked — the GitHub Release notes agree):
1. Remove or reword the `[Unreleased]` bullet — it describes already-shipped
   work, not pending work.
2. Note which tag it shipped in, if that context is useful for the
   surrounding CHANGELOG edit.

**Checks are inconclusive or contradictory** (no clear candidate commit, the
commit appears in no tag yet, or the GitHub Release notes disagree):
1. Do not remove the bullet — treat it as still `[Unreleased]`.
2. If the ambiguity itself seems worth recording, note it rather than
   guessing either way.

## Status

**First draft, single verified use.** This procedure was executed once, for
mission `20260831-release-v1014-docs-refresh`, to confirm several
`[Unreleased]` bullets had already shipped. It has not yet been confirmed to
generalize to a differently-shaped bullet or a different release cadence — a
second successful, independent application would raise confidence in
promoting it beyond first-draft status.

## Source

Originally captured as
`.analysis/pending/20260831-changelog-verification-runbook-request.md`, an
Opportunity Attack side quest (`verified_procedure_worth_preserving`,
`executable_decision_procedure`) from mission
`20260831-release-v1014-docs-refresh`.
