# ADR-0001 — Compiled artifacts in gzip+JSON with fast path

**Status:** Accepted  
**Date:** 2026-05-29  
**Context:** Migration to Go (bigbang-go-20260529)

---

## Context

The agent needs to load the skill configuration (active.yaml, personas, roles, domain templates) at the start of **every mission**. With separate YAML files, this loading involves multiple disk reads, YAML parsing, and structure merging — an operation repeated on every invocation.

The simplest alternative would be to read the YAMLs directly every time, without preprocessing.

## Decision

Compile the YAML sources into **gzip+JSON** artifacts stored in `.strategist/.compiled/`:

| Artifact | Sources |
|----------|---------|
| `.config.gz` | `active.yaml` + `personas/` + `roles/` |
| `.domain.gz` | `templates/domain/` |
| `.index.gz` | `knowledge.index.yaml` |
| `.manifest.gz` | SHA256 of the 3 artifacts above |

Each artifact includes a `sources: map[path → mtime]` field that allows staleness detection without recompiling.

The agent implements a **fast path**: if the artifact exists and is not stale (`check-stale`), it does `gunzip + JSON parse` instead of reading and parsing multiple YAMLs. The **standard path** (reading YAMLs directly) works as a fallback in case of corruption.

## Consequences

**Positive:**
- Configuration loading in a single operation (decompress + decode JSON)
- Corrupted artifact has automatic fallback to standard path — no mission stop
- `strategist check-stale` allows CI to verify whether recompilation is needed without loading anything
- Single format for all config types — no scattered merge logic

**Negative:**
- Requires `strategist compile` after manual YAML edits (an extra step that can be forgotten)
- `.strategist/.compiled/` must be in `.gitignore` — installation ensures this via `ensureGitignore`
- Two loading paths to keep in sync (fast path and standard path)
