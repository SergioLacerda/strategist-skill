# Discovery Artifact — Strategist Performance Optimization
**Mission ID:** perf-opt-20260529
**Task Type:** architecture_analysis
**Slot:** Ranger (discovery)
**Date:** 2026-05-29
**Sources:** strategist_performance_optimization.md + strategist_performance_opt2.md

---

## 1. Executive Summary

Two raw analyses were submitted addressing the same engineering concern: the Strategist skill's offline pipeline (bootstrap + preflight + context enrichment + learning phase) accumulates unnecessary latency through repeated YAML parsing and linear I/O on every mission invocation.

**Análise 1** (`strategist_performance_optimization.md`) diagnoses four bottleneck tiers and proposes a pre-compilation strategy using MessagePack + gzip to eliminate redundant parsing. It provides benchmark estimates, a phased roadmap, and Python pseudocode for a `StrategistCompiler`.

**Análise 2** (`strategist_performance_opt2.md`) is a direct follow-up clarifying that the proposed compilation is OS-agnostic serialization (not native binary), comparing it explicitly to Go/Rust compilation and concluding MessagePack is the right choice.

**Relationship:** The second analysis is an architectural decision record (ADR) for the first. Together they form one complete proposal. The correct treatment is to merge them into a single coherent specification before refinement.

[OBSERVATION] Neither analysis references actual profiled measurements. All timing figures (310ms baseline, 70ms target) are estimates derived from rule-of-thumb I/O costs, not from instrumented runs of the Strategist pipeline.

---

## 2. Inter-Module Boundaries

The following Strategist components are affected by the proposal:

| Component | Current Behavior | Proposed Change | Boundary Type |
|-----------|-----------------|-----------------|---------------|
| `install.sh` | Copies skill files, sets permissions | Must also run `compile-all.sh` | Entry point — external trigger |
| `bootstrap` (SKILL.md §1) | Parses `active.yaml`, `personas/*.yaml`, `roles/*.yaml` | Checks `.compiled/.config`; falls back to YAML | Config read path |
| `preflight` (SKILL.md §2) | Loads `index.yaml`, reads N identity/directives files | Checks `.compiled/.domain`; falls back to YAML | Domain load path |
| `context-enrichment` | Linear scan of `knowledge.index.yaml` by tag | O(1) lookup in `.compiled/.index` | Knowledge retrieval path |
| `learning-curator` | `outcomes.jsonl` write per mission | Buffered write (N missions or T seconds) | Persistence path |
| `compile-all.sh` | Does not exist | New: generates all `.compiled/` artifacts | Build-time tool — no runtime dependency |
| `.strategist/.compiled/` | Does not exist | New generated directory | Derived artifact store |

[OBSERVATION] The `learning-curator` buffer change is architecturally distinct from the serialization changes. It is a write-path optimization while Tiers 1–3 are read-path. Bundling them in one proposal risks coupling unrelated rollback paths.

### Boundary Invariants

- `.compiled/` artifacts are **derived** — the YAML source files remain the source of truth.
- `compile-all.sh` must be idempotent: re-running produces identical output if sources are unchanged.
- No component outside `install.sh` and the three read paths above should be aware that `.compiled/` exists.

---

## 3. Dependency Direction

```
YAML Sources (versioned)          Compiled Artifacts (derived, gitignored)
─────────────────────────         ──────────────────────────────────────────
knowledge.index.yaml      ──→     .compiled/.index   (msgpack.gz, inverted tag index)
index.yaml +                      .compiled/.domain  (msgpack.gz, all load_always +
  identity/*.yaml         ──→       load_by_task_type files as single blob)
  directives/*.yaml
  rubrics/*.yaml
active.yaml +                     .compiled/.config  (msgpack.gz, merged config blob)
  personas/*.yaml         ──→
  roles/*.yaml

.compiled/.manifest               SHA-256 checksums of all compiled artifacts
```

**Fallback contract:** If a compiled artifact is absent or stale (source mtime > artifact mtime), every read path falls back to the YAML source. No mission fails due to missing compiled artifacts.

[OBSERVATION] The stale-detection mechanism proposed (`compiled.stat().st_mtime > yaml_source.stat().st_mtime`) is insufficient. See §6 for specifics.

---

## 4. Risk Assessment

### 4.1 Technical Risks

| Risk | Severity | Likelihood | Detail |
|------|----------|-----------|--------|
| Stale compiled artifacts served after YAML edit | HIGH | HIGH | mtime-based check fails on network filesystems, FAT32, and when files are restored from backup with preserved timestamps |
| msgpack library version drift | MEDIUM | MEDIUM | No pinned version. Different agents using different msgpack versions may produce/read incompatible blobs |
| Cache invalidation gap for partial source edits | HIGH | MEDIUM | Editing one file in `identity/` does not invalidate `.compiled/.domain` unless all source mtimes are tracked |
| `__del__` flush in LearningBuffer is unreliable | MEDIUM | HIGH | Python `__del__` is not guaranteed to be called on interpreter exit (CPython) or called at all (PyPy). Outcomes could be lost on crash |
| Gzip decompression error not handled in fallback | MEDIUM | LOW | If `.compiled/` file is corrupt (partial write, disk error), fallback to YAML requires explicit error handling; pseudocode does not show it |

### 4.2 Portability Risks

| Risk | Severity | Detail |
|------|----------|--------|
| mtime comparison across timezones/DST | LOW | Python `stat().st_mtime` is UTC on POSIX; Windows has known FAT32 2s resolution issues |
| gzip availability on Windows | LOW | Python stdlib `gzip` is available; `bash strategist/compile-knowledge-index.sh` uses `gzip` CLI — may not be in PATH on Windows |
| yq/jq dependency in compile script | MEDIUM | `compile-knowledge-index.sh` requires `yq` and `jq`. These are not guaranteed present on target machines. The analysis does not specify a version requirement or installation check |

### 4.3 Maintenance Risks

| Risk | Severity | Detail |
|------|----------|--------|
| Two code paths diverge over time | HIGH | Every future change to bootstrap/preflight must be implemented twice: once in the YAML-reading path and once in the compiled-artifact path. This doubles the surface area for drift |
| `.compiled/` accidentally committed | MEDIUM | If `.gitignore` entry is missing or wrong, binary blobs enter version control. The analysis mentions gitignore but does not specify the exact entry |
| Learning buffer data loss on crash | HIGH | A crash between buffer fill and flush discards mission outcomes silently |

[RISK] The dual code path maintenance risk is the highest long-term structural risk. It is not addressed in either analysis.

---

## 5. Quality Gaps (World-Class Engineering Criteria)

### 5.1 Testing

[GAP] **No test strategy proposed.** World-class engineering requires:
- Unit tests for `compile_knowledge_index()` and `compile_domain()` covering: empty sources, missing files, malformed YAML, and Unicode paths.
- Integration test: `compile-all.sh` on a fixture workspace → preflight reads compiled artifacts → correct knowledge lookup.
- Regression test: modify a YAML source → confirm stale detection triggers fallback.
- LearningBuffer tests: verify flush on `buffer_size` reached, flush on `flush_interval_sec`, and data integrity on simulated crash (file left open).

### 5.2 Rollback Plan

[GAP] **No rollback procedure defined.** Required:
- Explicit procedure for removing `.compiled/` artifacts and forcing YAML fallback.
- Documented signal for "revert to pre-optimization state" without code changes.
- The fallback mechanism (check compiled → fall back to YAML) IS the rollback, but it is not called out as such and not tested as such.

### 5.3 Cache Invalidation Completeness

[GAP] The mtime-based stale check is described for single-file comparisons only. Not addressed:
- Multi-file domain blob: which source file's mtime is checked? All of them? The newest?
- `knowledge.index.yaml` references external paths — if a referenced doc changes, the index is not stale by mtime.
- `roles/*.yaml` additions (new file added, existing blob still has old mtime).

### 5.4 Observability / Telemetry

[GAP] **No observability plan.** In a world-class system, the optimization must be measurable. Required:
- Log line on cache hit vs. cache miss per compiled artifact (bootstrap, domain, knowledge).
- Timing spans for each read path (YAML vs. compiled).
- LearningBuffer: log flush events with count and latency.
- Without this, the 4.4x claim cannot be verified in production, only in benchmarks.

### 5.5 Benchmark Methodology

[GAP] The 310ms → 70ms benchmark is presented as fact but has no methodology:
- What harness produces these measurements?
- Were they measured on a specific machine/OS/disk type?
- Was it a cold start (no OS page cache) or warm?
- Are the timings inclusive of Python interpreter startup or exclusive?
- What is the confidence interval?

Without a reproducible benchmark, the 4.4x claim is an engineering estimate, not a measured result.

### 5.6 Security

[GAP] Compiled artifacts are binary blobs written during `install.sh`. Not addressed:
- If `install.sh` is run from an untrusted source, can a malicious `.compiled/` artifact be injected?
- The `.manifest` SHA-256 file is written by the same process that writes the artifacts — it cannot self-validate.
- Consider: `.manifest` should be signed or compared against a known-good hash from the YAML source.

---

## 6. Technical Specification Gaps

### 6.1 install.sh Integration Point

[GAP] The analysis states compilation happens "during install.sh" but does not specify:
- **Position:** Before or after dependency checks (`yq`, `jq`, `msgpack` presence)?
- **Error handling:** If `compile-all.sh` fails, does install.sh abort or continue with YAML fallback?
- **Re-install behavior:** Does re-running install.sh recompile unconditionally or only if sources changed?

**Required specification:**
```
install.sh execution order:
  1. Validate dependencies (yq, jq; msgpack optional)
  2. [If msgpack available] Run compile-all.sh
  3. [If compile-all.sh fails] Warn but continue — YAML fallback is guaranteed
  4. Write .gitignore entry for .compiled/
  5. [Existing install steps]
```

### 6.2 .gitignore Entry

[GAP] The analysis mentions "gitignore'd" without specifying the entry. Required:
```gitignore
# Strategist compiled artifacts (generated by install.sh, do not commit)
.strategist/.compiled/
```
The path must be relative to the target repo root, not the skill root. If `base_path` varies per workspace, the `.gitignore` entry must be templated or documented.

### 6.3 Stale Detection — Multi-Source

[GAP] For `.compiled/.domain`, which aggregates multiple YAML files:

**Proposed fix:** Track a manifest of source file mtimes inside the compiled blob, not just compare one mtime.

```python
# Inside compiled blob header:
{
  "compiled_at": 1748476800,
  "sources": {
    "identity/what-i-am.yaml": 1748470000,
    "identity/drift-patterns.yaml": 1748470001,
    "directives/core.yaml": 1748469000
  },
  ...
}

# On load:
def is_stale(compiled_blob, domain_root):
    for rel_path, recorded_mtime in compiled_blob["sources"].items():
        actual_mtime = (domain_root / rel_path).stat().st_mtime
        if actual_mtime > recorded_mtime:
            return True
    return False
```

### 6.4 msgpack Dependency

[GAP] Not specified:
- **Python:** `msgpack` PyPI package, minimum version? (`>=1.0.0` recommended — pre-1.0 had breaking API changes)
- **Bash compile path:** The `compile-knowledge-index.sh` uses `jq` to produce JSON, then gzip. It does NOT use msgpack in bash — so the compile path produces gzipped JSON, but the pseudocode reads msgpack. **This is an inconsistency in the analysis.**

[RISK] The bash compile script (`jq | gzip`) and the Python read path (`msgpack.unpackb`) are incompatible as written. The bash script produces gzipped JSON; `msgpack.unpackb` cannot read JSON. One of the two must change.

**Resolution options:**
- A: Compile to gzipped JSON (bash-friendly); read with `json.loads(gzip.decompress(...))` — no msgpack dependency.
- B: Compile to msgpack (requires a msgpack CLI or Python compile step); read with `msgpack.unpackb`. Requires `msgpack` in the dependency list.
- C: Keep bash for knowledge index (gzip JSON), Python for domain blob (msgpack). Inconsistent but functional.

**Recommendation:** Option A — gzipped JSON — eliminates the msgpack dependency entirely and is fully bash-agnostic. The performance difference between JSON and msgpack deserialization at this data size (<1MB) is negligible in the context of a 300ms→70ms optimization.

### 6.5 LearningBuffer — Flush Guarantee

[GAP] The `__del__` method for guaranteed flush is not reliable:
- In Python, `__del__` is called on garbage collection, which may not happen at interpreter shutdown.
- On unhandled exceptions, `__del__` may or may not be called depending on the exception path.

**Required:** Use `atexit.register(buffer.flush)` in addition to or instead of `__del__`:
```python
import atexit

class LearningBuffer:
    def __init__(self, ...):
        ...
        atexit.register(self.flush)
```

---

## 7. Modules Index

| Module | File(s) | Status | Boundary Defined? |
|--------|---------|--------|-------------------|
| bootstrap | SKILL.md §1, active.yaml | Existing | YES — reads config only |
| preflight | SKILL.md §2, index.yaml | Existing | YES — reads domain only |
| context-enrichment | SKILL.md §4, knowledge.index.yaml | Existing | YES — reads knowledge only |
| learning-curator | SKILL.md §8, memory/outcomes.jsonl | Existing | PARTIAL — write boundary not defined |
| compile-all.sh | Does not exist | Proposed | NOT DEFINED |
| compile-knowledge-index.sh | Does not exist | Proposed | NOT DEFINED |
| compile-domain.sh | Does not exist | Proposed | NOT DEFINED |
| StrategistCompiler (Python) | Does not exist | Proposed | NOT DEFINED |
| LearningBuffer (Python) | Does not exist | Proposed | PARTIAL — flush guarantee missing |
| .strategist/.compiled/ | Does not exist | Proposed artifact store | NOT DEFINED |

[OBSERVATION] Five of the ten modules in this proposal are undefined. A world-class engineering spec must define input/output contracts, error conditions, and ownership for all modules before implementation begins.

---

## 8. Summary of Findings

| Category | Finding | Severity |
|----------|---------|----------|
| Core proposal validity | MessagePack/gzip serialization is a sound approach | — |
| ADR completeness | Analyses 1+2 together form a complete ADR; merge recommended | LOW |
| bash/Python inconsistency | Compile script (JSON) ≠ read path (msgpack) | HIGH |
| Stale detection | Multi-source mtime tracking not specified | HIGH |
| LearningBuffer flush | `__del__` unreliable; atexit required | HIGH |
| Benchmark methodology | 4.4x claim lacks reproducible harness | MEDIUM |
| Testing | No test strategy | HIGH |
| Rollback | Implied but not documented | MEDIUM |
| Observability | Absent | MEDIUM |
| Security | Artifact injection not addressed | MEDIUM |
| Dependency spec | yq, jq, msgpack versions unspecified | MEDIUM |
| .gitignore | Entry not specified | LOW |

**Bottom line:** The optimization direction is correct and the ROI is real. The proposal cannot move to implementation as-is. Seven specification gaps must be resolved in refinement before any code is written.
