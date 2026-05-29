# Proposal — Strategist Performance Optimization
**Mission ID:** perf-opt-20260529
**Artifact:** proposal.md (what and why)
**Date:** 2026-05-29
**Status:** refined

---

## What

Pre-compile the Strategist skill's static YAML configuration into gzipped JSON artifacts at install time, so that every mission invocation can skip repeated YAML parsing and perform O(1) lookups instead of O(n) linear scans.

Three read paths are optimized:

| Read Path | Current | Optimized | Artifact |
|-----------|---------|-----------|----------|
| Bootstrap (§1) | Parse active.yaml + personas/*.yaml + roles/*.yaml | Decompress `.compiled/.config.gz`, parse JSON | `.compiled/.config.gz` |
| Preflight (§2) | Load index.yaml + N identity/directives files | Decompress `.compiled/.domain.gz`, parse JSON | `.compiled/.domain.gz` |
| Context enrichment (§4) | Linear tag scan over knowledge.index.yaml | O(1) key lookup in inverted index | `.compiled/.index.gz` |

A fourth change optimizes the write path:

| Write Path | Current | Optimized |
|------------|---------|-----------|
| Learning phase (§8) | Write one line to outcomes.jsonl per mission | Buffer N lines in a temp file; flush to outcomes.jsonl when threshold is reached |

---

## Why

### Constraint: Language-agnostic, shell-first

The original analyses proposed MessagePack as the serialization format and Python as the read/write path. Both are rejected.

**Decision:** All compilation and all decompilation of artifacts must be executable by shell scripts (`sh` on POSIX, `PowerShell` on Windows) with no runtime language dependency (no Python, no Node, no Go).

**Rationale:**
- The Strategist skill is installed into target repositories with no assumptions about available runtimes beyond POSIX shell and common Unix tools (`jq`, `yq`, `gzip`).
- Requiring Python or any other language runtime creates an install gate that conflicts with the skill's "standalone by default" design philosophy.
- gzipped JSON is fully readable by `jq` (parse) and writable by `jq | gzip` (compile). No additional library is needed.
- Performance difference between JSON and MessagePack at this data size (<1MB) is negligible relative to the 300ms→70ms target gain.

### Why the optimization is worth doing

Every Strategist mission invocation reads the same static configuration from disk. This configuration changes only when the skill itself is updated — which happens at install time. Reading and parsing it on every mission is pure waste.

The compiled artifacts trade:
- ~150–300ms of YAML parsing + linear search per mission
- for ~10–20ms of gzip decompress + JSON parse per mission
- with a one-time ~500ms compilation cost at install time

At 10 missions per session, this saves ~2–3 seconds of offline latency per session. At 100 missions, it saves ~25 seconds.

### Benchmark note

These figures are engineering estimates. No instrumented measurement of the actual pipeline exists. A separate benchmarking analysis (or test analysis) should establish a baseline and confirm the gain post-implementation.

---

## What This Proposal Covers

| In scope | Out of scope |
|----------|-------------|
| Compile scripts for `.index`, `.domain`, `.config` | Tests (→ separate analysis) |
| Agent instructions for reading compiled artifacts | Benchmark harness (→ test analysis) |
| LearningBuffer in shell (write path) | Security hardening of compiled artifacts |
| install.sh integration | Native binary (Go/Rust) compilation |
| Windows compatibility (PowerShell path) | Runtime performance of Claude API calls |
| Stale detection via embedded source manifest | |
| Rollback procedure (delete `.compiled/`) | |

---

## Contract Standardization — Separate Analysis Recommended

During refinement, five of the ten modules involved in this proposal were found to have no formal contracts (input/output spec, error conditions, ownership). These are all new modules introduced by this proposal.

Additionally, the existing Strategist modules (`bootstrap`, `preflight`, `context-enrichment`, `learning-curator`) have implicit contracts embedded in SKILL.md prose that have never been formally specified, making them difficult to maintain and test.

**Recommendation:** Create a separate analysis (`contracts-standardization`) to formally specify contracts for **all** Strategist modules — both existing and new. This analysis should produce:
- A `contracts/` directory in `.strategist/` with one contract file per module
- A contract schema (input, output, error conditions, write scope, ownership)
- Updated references in SKILL.md pointing to contract files

This proposal defines the contracts for the four new shell scripts and three compiled artifact schemas (in `design.md`). Those definitions should be the first entries in the contracts analysis when it is created.

---

## Decision Record

| Decision | Rationale |
|----------|-----------|
| gzipped JSON over MessagePack | Shell-readable with `jq`; no library dependency |
| Shell scripts over Python for compile path | Language-agnostic; no runtime dependency |
| Stale detection via embedded source manifest | mtime of single file is insufficient for multi-file blobs |
| LearningBuffer in shell (temp file + flush) | Eliminates Python `__del__`/`atexit` reliability gap |
| Testing deferred | Separate analysis per user direction |
| Rollback = delete `.compiled/` | Fallback to YAML is always guaranteed; no code change needed |
