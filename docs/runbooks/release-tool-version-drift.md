# Runbook: A release breaks on a run where nothing changed (tool-version drift)

**Shape:** procedural (see `README.md` — `Trigger` / `Steps` / `Decision Point` /
`Stop Conditions` / `Reference`).

**Scope boundary — read this first.** This runbook documents a *class* of failure:
CI config whose validity depends on a tool version that can change without a commit.
It is **not** the post-mortem of the `v1.0.8` / `v1.0.9` release failures. For that
incident the tool-version hypothesis was explicitly tested and ruled out
(`goreleaser-action v6.4.0 → v7.2.3`, same symptom on `v1.0.9`); its proximate cause
was an artifact path/layout mismatch, covered by `release-no-dist-artifacts-found.md`.
Do not read the two as the same problem.

## Trigger

Any one of:

- A tag-triggered release fails on configuration that worked for the previous tag,
  and `git log` shows no change to the failing config or workflow.
- You are about to bump, pin, or unpin a tool version anywhere under
  `.github/workflows/`.
- A deprecation warning appears in a CI log for a config file nobody edited.
- You are adding a new tool to the pipeline and choosing how to express its version.

## Steps

### 1. Classify the tool before touching anything

The class determines everything that follows. There are four in this repository:

| Class | How the version is expressed | Dependabot | Failure mode |
|---|---|---|---|
| **A** | SHA pin + `# vX.Y.Z` comment on `uses:` | sees it | none — pinned *and* maintained |
| **B** | literal version in a `with:` input or `go install @vX` | **blind** | silent freeze; drift by neglect |
| **C** | floating range (`~> v2`, `^1`, `latest`) | **blind** | **changes with no commit** |
| **D** | derived from a repo file (`go-version-file`) | n/a | none — single source of truth |

Current membership:

- **A** — all 13 `uses:` references across `release.yml`, `test.yml`, `pages.yml`,
  `codeql.yml`. Dependabot's `github-actions` ecosystem bumps these weekly
  (`.github/dependabot.yml`).
- **B** — `golangci-lint-action` `version: v2.12.2` (`test.yml`), and
  `go install golang.org/x/vuln/cmd/govulncheck@v1.1.4` (`test.yml`). Also
  `node-version: '22'`, hardcoded in both `test.yml` and `pages.yml` while
  `web/landing/package.json` declares `engines.node >= 22.12.0` — two literals and a
  range with nothing enforcing agreement. No incompatibility has been observed;
  the risk is the duplicated source of truth, not a current break.
- **C** — `goreleaser/goreleaser-action` `with.version: "~> v2"` (`release.yml`).
  **This is the only member.**
- **D** — every Go job's `go-version-file: "go.mod"`. `go.mod` declares the `go` and
  `toolchain` directives; the workflows never restate them.

Note the asymmetry: **every version Dependabot can see is pinned and actively
maintained; every version it cannot see is either frozen by hand or floating.**
Dependabot's `github-actions` ecosystem reads the `uses:` line only. It does not read
`with:` inputs, `go install` arguments, or `node-version`. `gomod` reads
`go.mod`/`go.sum` only. Class B and Class C versions live in neither place.

### 2. Class C first — it is the only one that moves on its own

A floating range means the binary that runs today is not necessarily the binary that
ran last week, with no commit to point at. Two properties of this repository make
that sharper than usual:

- The **only** Class C member is GoReleaser — the tool that produces every release
  artifact.
- It runs in `release.yml`, which triggers only on `push: tags: ['v*.*.*']`. Nothing
  validates `.goreleaser.yaml` on a branch or a PR.

Together: a config field that is valid today can stop being valid on a run where the
repository did not change, and you will find out **after** a tag exists — often after
a GitHub Release has been partially created. That is the most expensive possible
moment to discover it.

This is not hypothetical for this repository. `.goreleaser.yaml` carried the singular
`archives[].format:` field, which GoReleaser v2 replaced with the plural `formats:`
list; the singular form is accepted with a deprecation notice. A deprecated field
under a floating resolver is exactly the compound condition above. It has since been
changed to `formats: [binary]`.

### 3. Validate the config against the version that will actually run

Do not reason about which version introduced or removed a field, and do not write a
version number into a comment as a substitute for checking. Run the tool's own
validator:

```bash
goreleaser check                      # validates .goreleaser.yaml, reports deprecations
goreleaser release --snapshot --clean # full local build, publishes nothing
```

`goreleaser check` is correct against every future version. A version number written
into a runbook or a comment goes stale silently.

If you cannot install the tool locally, this validation belongs in CI (see the
Decision Point below) — not in your head.

### 4. Audit the Dependabot blind spot

Class B versions are deterministic, which is the desirable half. They are also
invisible to every configured ecosystem, so they stay frozen until a human edits them
by hand. Grep for them before assuming the pipeline is current:

```bash
grep -rn "version:" .github/workflows/ | grep -v "go-version-file"
grep -rn "go install .*@v" .github/workflows/
grep -rn "node-version:" .github/workflows/
```

Anything these return is on nobody's update schedule. This repository has no
compensating control for that today.

### 5. Prefer Class D wherever the tool supports it

`go-version-file: "go.mod"` is the pattern to copy: the version lives in one file the
build already depends on, and the workflow never restates it. When adding a tool, ask
whether it can read its version from a repo file before writing a literal into a
workflow.

### 6. Release-time hooks must not mutate dependency versions

`.goreleaser.yaml`'s `before.hooks` ran `go mod tidy`, which can rewrite
`go.mod`/`go.sum` for the very commit being tagged — a dependency-version mutation
performed *during* the release. It has been replaced with `go mod download` +
`go mod verify`. Module tidiness is already enforced on every push by `test.yml`'s
"Module hygiene" step (`go mod tidy` followed by `git diff --exit-code`), so the
release does not need to re-derive it.

General rule: a release may *verify* dependency versions. It must not *change* them.

## Decision Point

**How do you stop Class C from surprising you?** This repository has not decided.
Both options mutate `.github/workflows/`, and the choice is a durable supply-chain
policy, so it deserves an explicit owner rather than a default.

| | **A. Pin an exact `2.x.y`** | **B. Keep `~> v2`, add a PR-time `goreleaser check`** |
|---|---|---|
| Determinism | full — same bytes on every rebuild | executed version still varies per run |
| Upstream fixes | wait for a human to bump | arrive automatically |
| Deprecations | surface whenever you bump | surface on a PR, not on a tag |
| Cost | manual bumps forever, no Dependabot path | the check must be pinned to match the release |

**They are not mutually exclusive.** Pinning exactly *and* running `goreleaser check`
on PRs gives determinism plus early warning when you do bump. Choose deliberately;
record the choice as an ADR rather than leaving it implied by the workflow file.

Whichever you pick, the structural gap behind it is that `release.yml` is never
exercised before a real tag exists. Closing that gap — running `goreleaser check`
and/or `goreleaser build --snapshot` on branches and PRs, pinned to the same version
the release resolves — is what converts this whole class from "found at tag time" to
"found at PR time."

## Stop Conditions

- **A tag already exists for the failing run.** Do not simply re-run the workflow.
  Follow `release-asset-already-exists.md` first — a partially-created release changes
  what a re-run does.
- **The symptom is missing or mismatched `dist/` artifacts.** That is a different
  class. Follow `release-no-dist-artifacts-found.md`; do not treat it as version
  drift.
- **`goreleaser check` cannot confirm the version relationship.** Stop and report. Do
  not guess a version number into the config, and do not widen a floating range to
  make an error disappear.
- **The fix requires editing a workflow and you are operating under a
  documentation-only scope.** Stop and hand off — every remedy in the Decision Point
  mutates `.github/workflows/`.

## Reference

- Class inventory and evidence with file/line citations:
  `.analysis/refined/20260729-ci-versioning-runbook/analysis.md` (F1–F9)
- Shape rationale, the resolved uncertainties, and the pin-vs-float trade-off:
  `.analysis/refined/20260729-ci-versioning-runbook/design.md` (D1–D8)
- Ruled-out version hypothesis for the `v1.0.8`/`v1.0.9` incident:
  `.analysis/refined/20260729-goreleaser-ci-diagnostic-v2/design.md`
- Sibling runbooks, different classes: `release-asset-already-exists.md`
  (duplicate release assets), `release-no-dist-artifacts-found.md`
  (artifact path/layout mismatch)
- Analysed 2026-07-29. No GoReleaser version number is asserted for the
  `format:`/`formats:` deprecation — that relationship was not verifiable from the
  workspace at the time of writing, and `goreleaser check` is the authoritative test
  regardless.
