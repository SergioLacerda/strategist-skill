# Runbook: Strategist philosophy and release evidence

Use this runbook when a change claims to close an architecture, provider, or
release residual. The claim must identify the evidence layer; no single green
check proves every layer below.

## Evidence ladder

1. **Source and generated parity** — authoring defaults, embedded artifacts, and
   installed `.strategist` files agree.
2. **Local validation** — focused unit tests, spec/integration tests, `go vet`,
   formatting, and reproducibility checks pass in an isolated environment.
3. **Snapshot release evidence** — GoReleaser produces the expected local
   binaries, checksums, signatures, and SBOM inputs.
4. **Published release evidence** — the concrete Git tag and GitHub Release
   expose the expected assets and their signatures/attestations.
5. **Installation evidence** — a clean target installs and validates the
   selected artifact without relying on the developer's HOME, caches, or
   provider state.

Local source, test, or snapshot evidence must never be described as proof of a
published GitHub Release or a live external provider. A release preparation
document is a checklist and evidence boundary, not a release claim.

## Authority vocabulary

- `Role` is the fixed Strategist responsibility (`Ranger`, `Archivist`, or
  `Sniper`); `Provider`/weapon is the replaceable package used within that role.
- `execution` is the pipeline slot and phase; `materialization` is the bounded
  write operation performed by the current Sniper contract.
- Product version (`v1.0.17`) is distinct from a skill/package contract version
  (`skill-package/v1`) and from a provider package version.
- The executable routine name is `opportunity_attack`; `opportunist_attack` is
  not a valid runtime identifier.

## Stop conditions

Stop and report a residual when the selected release commit is dirty, generated
parity drifts, a required asset/signature/attestation is absent, a test relies
on host HOME or an uncontrolled cache, or a catalog/lock record is being used
as a substitute for live provider evidence.
