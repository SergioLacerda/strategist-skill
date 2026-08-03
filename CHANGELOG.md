# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## Release History Authority

`CHANGELOG.md` is the curated source for unreleased changes and the historical
`1.0.0` baseline. For patch releases after `1.0.0`, GitHub Releases are authoritative
for published release notes and downloadable assets. Do not
backfill `v1.0.x` notes into this file unless they are verified against the
corresponding tag and GitHub Release.

---

## [Unreleased]

### Security

- **BREAKING:** `OTEL_EXPORTER_OTLP_INSECURE` now defaults to `false` (TLS required).
  If you send telemetry to a plaintext gRPC endpoint, add `OTEL_EXPORTER_OTLP_INSECURE=true`
  to your environment. This change closes a security finding where telemetry data could be
  intercepted on networks without TLS.
- Release binaries are now signed with cosign keyless (Sigstore). `.bundle` files are uploaded
  alongside each binary for offline verification via `cosign verify-blob`.
- Config integrity check: CLI emits a warning to stderr when `active.yaml` is modified
  outside the CLI. Lock file written at `install` time; checked on every CLI startup.
- Added `SECURITY.md` with responsible disclosure policy, release verification instructions,
  and recommended branch protection settings.

### Added
- SHA256 checksum verification in `bootstrap.sh` for versioned releases
- Security warning when installing from branch ref without integrity check
- Install rollback mechanism in `strategist/install.sh` (`INSTALL_MANIFEST` + `trap ERR`)
- YAML config validation step in Strategist preflight (`2a.validate`)
- New schemas: `active.schema.yaml`, `roles.schema.yaml`, `slot-output.schema.yaml`
- Slot output contract validation after Ranger and Archivist phases
- Test harness with 5 golden-file fixtures for critical contract scenarios
- CI workflow `test.yml` with shellcheck, fixture tests, and schema validation
- `SHA256SUMS` asset generation in `release.yml`

### Changed
- `protocol.md`: normalized `risk_score` vocabulary to `write_pending` / `write_analysis` / `controlled`
- `readme.md`: added security callout for curl pipe installation

---

## [1.0.0] - 2026-05-28

### Added
- Core compilation scripts and contract definitions for strategist configuration and indexing management
- Dungeon documentation pages and GitHub Pages deployment workflow (`pages.yml`)
- Architecture and integration flow diagrams (`docs/`)
- `validate_provider()` function and context hints to install wizard
- Slot write contracts: `write_pending` (Ranger) and `write_analysis` (Archivist)
- `opportunity_attack` phase and side quest pipeline (phases 5b–5d)
- Design spec for side quest ataque de oportunidade pipeline
- Design spec for slot risk contract fix and known-providers registry
- Curl installer (`bootstrap.sh` / `bootstrap.ps1`) with GitHub Actions release workflow (`release.yml`)
- Design spec for curl installer and GitHub Actions release CI/CD
- `install.sh` generates `.strategist/` runtime and registers agent shims across Claude, Gemini, Codex
- Implementation plan and design spec for multi-agent skill registration via `.strategist install`
- `.analysis/` workspace directories tracked in git (`pending/`, `refined/`, `archived/`)

### Changed
- Renamed core persona roles: Scout → Ranger, Engineer → Archivist, Hunter → Sniper
- Removed legacy analysis directory and simplified bootstrap installation flow
- Replaced verbose documentation with concise technical overview of mission orchestration

### Fixed
- Pinned all GitHub Actions to commit SHA to prevent supply-chain attacks
- Set executable permission on `bootstrap.sh` and `bootstrap.ps1`
- `.gitignore` updated to allow tracking `.analysis/pending/`, `refined/`, `archived/`

### Removed
- Obsolete strategist-mission-pipeline design docs and specifications
- Pending design specs (superseded by implementations)
