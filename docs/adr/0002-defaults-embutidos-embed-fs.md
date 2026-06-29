# ADR-0002 — Defaults embedded in the binary via embed.FS

**Status:** Accepted  
**Date:** 2026-05-29  
**Context:** Migration to Go (bigbang-go-20260529)

---

## Context

The `strategist install` command needs to copy ~60 files (SKILL.md, personas, roles, schemas, contracts, templates) to the `.strategist/` directory of the target repository. These files must be available in any environment where the binary is executed.

Alternatives considered:
- **Network fetch** — download files from GitHub at install time
- **External bundling** — distribute a tarball alongside the binary
- **embed.FS** — embed the files directly in the binary at compile time

## Decision

Embed all defaults in `internal/embed/defaults/` using `//go:embed all:defaults`. The `embed.Extractor` package implements `domain.FileExtractor` and copies the embedded FS to disk via `fs.WalkDir`, preserving the directory structure.

```go
//go:embed all:defaults
var defaultsFS embed.FS
```

`internal/embed/defaults/` is an exact copy of `strategist/` — any change to the skill runtime must be reflected in both.

## Consequences

**Positive:**
- `strategist install` works **offline** and without external dependencies (no curl, jq, git)
- Bootstrap via `curl | bash` downloads a single self-contained binary — no separate assets
- No silent network failures at install time
- Defaults version is fixed to the binary version — no drift between binary and runtime

**Negative:**
- `strategist/` and `internal/embed/defaults/` must be kept in sync — drift is detectable via diff, but not automatically blocked in CI
- Binary grows with the embedded defaults (~a few KB of compressed YAML)
- Editing defaults requires recompiling the binary — it is not possible to update only the YAML files in production without a new release
