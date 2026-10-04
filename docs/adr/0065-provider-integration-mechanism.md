# ADR-0065 — Provider Integration Mechanism

**Status:** Accepted (core implemented; consumers pending)
**Date:** 2026-10-04
**Mission:** `20261004-integration-mechanism-jev` (refined package, wave W0)
**Related:** [ADR-0064](0064-canonical-seven-family-taxonomy.md), [ADR-0050](0050-confidence-governance-contract.md), [ADR-0061](0061-weapon-sidecar-generation-and-versioned-catalog.md)

## Context

Strategist needs to let a consumer delegate a *check* to an external provider and still
work, unchanged, when that provider is absent. The first provider is JEV, an HTTPS
typed-question evaluator (`POST /v1/systemone`, bearer credential, `noul`, `choice` and
`score` questions, answers with `confidence` and `usage`). The first consumer is the HANDOFF
validation, which today runs entirely locally (`main`).

The repository had no HTTP transport, no credential reference convention and no place for an
operator's integration decision. It also has a rule that a classification follows
responsibility: a Mechanism is a deterministic rule, a Tool is an operation that returns a
result ([ADR-0064](0064-canonical-seven-family-taxonomy.md)).

## Decision

1. **One Mechanism, `provider_integration`**, owned by `internal/integration`. It decides,
   deterministically and per attempt, whether the provider or `main` serves a call, and it
   records the reason. The call itself is delegated to an internal transport component, so
   the Mechanism stays a rule and the operation stays inside it. The split into a Tool plus a
   Mechanism remains the documented alternative if the transport ever gains a second client.
2. **Identifier.** `provider_integration` replaces the placeholder `integration_runtime`:
   "runtime" already names the generated `.strategist/` instance and the Ranked runtimes.
3. **Preference and fallback.** JEV is preferred when enabled, bound, allowed, credentialed,
   within its data policy and healthy; every other state selects `main`, so the provider never
   blocks a handoff. That includes `binding_integrity_failed`: `main` serves the call and runs
   every deterministic check, the decision is flagged suspect so the consumer reports it, and
   the tampered binding never serves a call. `disabled` is a configured choice, not a fallback.
4. **A provider answer is data.** Only a local rule decides. Integrity, Approval Gate
   evidence, authorization and policy facts are never delegated. A delegated check is
   limited to input and output conformance to a handoff contract's criteria.
5. **Adapters resolve by explicit provider and exact version** from a compile-time registry.
   Catalog order, heuristics and "latest" never select one. The resolved provider model is
   part of the binding identity.
6. **Operator configuration** is `.strategist/integrations.yaml`
   (`strategist-integrations/v1`): a credential *reference* (`env:` or `dotenv:`), a plain
   https endpoint, a pinned model, allowed capabilities and a data policy. It is written
   atomically with owner-only permissions and preserved by upgrade (verified by test).
   Strategist never stores the secret.
7. **Credentials** are parsed in memory and never exported to the process environment,
   because host bridges build child environments from `os.Environ()`.
8. **Enforcement tier.** The row is `machine_observed`: the Mechanism runs on a reachable
   path (the two lifecycle handoff evaluations) and reports through command output and a
   call ledger, but never blocks. It was `agent_only` until the HANDOFF consumer was wired,
   per the `errors.yaml` rule that code with no live caller is not observed.
9. **First consumer.** The HANDOFF validation pre-checks input and output conformance at
   `ranger_to_archivist` and `archivist_to_sniper`. One aggregate question carries the
   confidence criterion; the fixed local rule approves only strictly above the threshold
   (default 0.90). An approval is recorded as a `delegation` on the handoff outcome
   ([handoff-contract](../../internal/embed/defaults/contracts/machine/handoff-contract.yaml)
   `delegated_validation`); anything else leaves the handoff on `main`.

## Consequences

- Positive: JEV is optional and its failure never blocks a handoff; the base is
  provider-agnostic; secrets stay out of artifacts, logs, locks and child processes.
- Negative: three new registry artifacts (this ADR, the contract file, the registry row) and
  a taxonomy-inventory entry to keep in sync; behavior of the real JEV service beyond its
  documented examples is unknown and is treated defensively (single retry, local size limits).
- Not decided here: the wizard and `integrations doctor`, the ROSTER binding and the
  PRECISE-SHOT and SHARPSHOOTER work. Those are separate, later waves.

## Evidence

Documented JEV shapes are taken from the official quickstart; nothing was verified against a
live call. The temporary test key file was never opened.
