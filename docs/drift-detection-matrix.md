# Drift Detection Matrix

**Status:** Accepted
**Last Updated:** 2026-08-30

This doc maps every existing deterministic drift/consistency detector in this
repository to the drift class(es) it actually covers, so a passing check (or
a telemetry line that says "no drift", "unmodified", "fresh", "OK") stops
implying more than it verified. It was produced by
`.analysis/refined/20260830-skill-gaps-triage/` Task 5 (G17), closing K08's
finding: the detectors already existed and were individually documented, but
nothing classified them against a shared taxonomy.

## Taxonomy

Six drift classes, from most literal to most interpretive:

| Class | Question it answers |
|---|---|
| `byte` | Is the raw content identical to a known-good copy? |
| `schema` | Does the content have the right structural shape (required fields, valid types)? |
| `provenance` | Does this artifact's lineage trace back to a known, trusted source or prior state? |
| `contract` | Does a declared interface/value stay consistent across the artifacts that reference it? |
| `behavior` | Does the running system actually do what the spec says at invocation time? |
| `semantic` | Is the content *correct/meaningful*, beyond being structurally valid and consistent? |

A single detector commonly covers more than one class at once (e.g. a byte
comparison can also serve as a provenance check when the "known-good copy" is
itself a provenance-tracked manifest entry).

## Per-Detector Classification

| Detector | Classes covered | Why |
|---|---|---|
| `scripts/check-contract-consistency.sh` | `contract`, `byte` | Greps for literal declared-value substrings (e.g. `contract: write_analysis`, a specific Markdown table row) across `skill.yaml`, `docs/*.md`, and `tests/spec/*.feature`, asserting they stay in sync. Each individual assertion is a raw substring/byte match; the thing being verified across files is contract consistency, not structural shape or meaning. |
| `scripts/check-convergence.sh` | `contract`, `byte` | Same mechanism (literal `grep -q` on exact code/doc strings) applied to a different invariant: that the runtime/package-boundary migration (retiring the root `strategist/` authoring mirror, canonicalizing `skills/<provider>/skill.yaml`) hasn't regressed anywhere it's referenced. |
| `scripts/check-docs-governance.sh` | `schema` | Checks structural requirements of the `docs/` corpus itself: every `docs/*.md`/`docs/adr/*.md` has required status/date frontmatter fields, no relative parent-directory links out of the docs tree, no unfinished-placeholder markers, and every doc is reachable from `docs/README.md`'s navigation table. This is document-shape validation, not a check that any doc's *content* is correct. |
| Golden test suite (`tests/evals/golden/`, ADR-0026) | `byte`, `schema` | Three comparator modes in `Assert`/`Canonicalize` (`golden.go`): `Exact` and `Normalized` (canonicalize volatile fields — timestamps, UUIDs, temp paths, hashes — then byte-compare) are `byte`; `Structural` (used for `compiled-contract-shape`) parses and compares shape rather than raw bytes, i.e. `schema`. One subtest, `cli-help`, is the only detector in this matrix that invokes a live binary (`go run ./cmd/strategist --help`) and compares its actual stdout — the closest thing to a `behavior` check here, though the comparison itself is still `Exact` byte matching against a fixture, not an assertion about what the output *means*. |
| `strategist check`'s hash/fingerprint comparisons | `byte`, `provenance` | Two independent mechanisms: `internal/integrity` compares `active.yaml`'s current SHA256/size/mtime against the fingerprint sealed in `.config.lock` at the last CLI-trusted write (`ReasonUnmodified` = no drift) — byte identity plus provenance (did this file change since the CLI itself sealed it, not just "is it byte-different from something"). `internal/check/check_runtime.go#validateRuntimeDefaultParity` compares each on-disk runtime file's exact bytes against the embedded default and, on mismatch, classifies via `domain.RuntimeDefaultDecision` (`auto_upgrade` if the on-disk SHA256 matches the install manifest's recorded original, `conflict` otherwise) — byte comparison first, provenance classification second. |
| `strategist check-stale` (`internal/stale`) | `byte`, `provenance` | `checkManifest` compares a compiled artifact's current SHA256 against the value recorded in `.manifest.gz` at compile time (byte). `checkArtifactSources` compares each declared source's current mtime/size against what was recorded when the artifact was compiled, flagging `source_newer` when a source outran its derived artifact — a provenance/lineage check (is the compiled artifact still a faithful derivative of its declared sources), not a byte or schema check on the artifact's own content. |

## The Gap This Matrix Makes Visible

**No detector in this repository covers `behavior` in the general sense**
(does the running system actually produce the right *effect* for a given
input across the range of cases the spec describes) **or `semantic` at all**
(is the content correct/meaningful, not just structurally valid and
byte-consistent). This is the same finding K08 in
`.analysis/refined/20260830-skill-gaps-triage/analysis.md` Cluster 4
identified: the detector diversity already exists, it was just unmapped, and
mapping it makes the semantic gap visible instead of implicit. The golden
suite's `cli-help` subtest is the sole partial exception — it captures one
concrete slice of live behavior (CLI help text) — but even that is checked by
exact byte comparison to a frozen fixture, not by evaluating whether the
output is correct for arbitrary new inputs.

This gap is intentional to leave open, not a defect to silently patch here:
closing it (e.g. a semantic-eval layer, per this mission's own Task 12 /
Dojo findings G13–G14) is a larger, separate investment that this matrix's
job is only to make legible, not to fund.

## Cross-References

Code and scripts that emit a "no drift" / "unmodified" / "fresh" / "OK"
result carry an inline comment pointing back to this table:

- `internal/stale/result.go` (`ReasonFresh`)
- `internal/integrity/warning.go` (`ReasonUnmodified`)
- `internal/check/check_runtime.go` (`validateRuntimeDefaultFile`'s byte-identity branch)
- `scripts/check-contract-consistency.sh`, `scripts/check-convergence.sh`, `scripts/check-docs-governance.sh` (their final `OK:` echo lines)

See also `docs/adr/0005-slot-write-contracts.md`'s Addendum and
`internal/embed/defaults/contracts/machine/errors.yaml` for the related but
distinct `machine_enforced`/`machine_observed`/`agent_only` vocabulary —
that one classifies whether an individual *contract clause* is machine-
checked at all; this matrix classifies what *kind* of drift an existing
detector checks for once it does run.
