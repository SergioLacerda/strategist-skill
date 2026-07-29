# Runbook — Deep Analysis Pass 4: World-Class Engineering Scorecard

Score the project 1–5 per dimension, **evidence first**: every score cites files/commands;
every gap cites the pass that found it or a fresh probe.

## Trigger

Pass 0 inventory complete; richer when Passes 1–3 exist (their findings become scored
evidence), but runnable standalone.

## Dimensions & probes

### 1. Contract–code parity
- Are documented rules enforced or tested? (contract-test YAMLs + their runner; FSM in
  code; error tokens present in code — Pass 0 §3.)
- Coverage: contract files vs contract-test files; do alignment tests validate their own
  `subject:` paths?
- Is the enforced corpus itself consistent? (Pass 2 S1 count caps this score.)

### 2. Testability
- Layer count (unit / integration / spec / scenario), `_test.go` count, race detector in CI.
- Blind spots: zero-test error families (Pass 0), mechanics without scenarios (Pass 3),
  divergences tests cannot catch because no machine representation exists (Pass 1).

### 3. Observability
- Telemetry surface (tracer/metrics/events), event invariants per phase, envelope
  contracts.
- Probe: pick every *degraded* mode (identity missing, registry incomplete) and ask —
  does it reach the operator, or is it a one-shot warn? Silent degradation caps this score.

### 4. Versioning & migration
- Artifacts: schema_version fields, generated headers, install manifest/lock, upgrade
  e2e tests, staleness tooling, migration commands, ADR count.
- Probe: list **unfinished migrations** — legacy tokens still normative anywhere,
  "future rename" declarations with no tracked plan, compiled artifacts that regenerate
  known defects (no template lint).

### 5. Security / supply chain
- Present: code scanning, release pipeline, integrity checks, knowledge trust model
  (tiers, promotion gates).
- Probe: is integrity blocking or warn-only? Is there an artifact signing/SBOM story or
  an open backlog item (link it, don't duplicate)?

### 6. Developer experience
- Docs estate breadth, CLI remediation hints in error messages, onboarding path.
- Probe: walk the **agent's** entry path (the primary user mid-mission) — startup token
  cost, broken pointers, undocumented internal components. Broken pointers on the entry
  path cap this score.

## Scoring discipline

- 5 = exemplary and consistent; 4 = strong with targeted gaps; 3 = real machinery,
  systemic gap; 2 = partial; 1 = absent.
- Overall = judgment call, not average — name the bottleneck dimension explicitly.
- Cross-dimension "top gaps" section: the 3–5 fixes that raise multiple dimensions at once.

## Decision Point

Done when `<base_path>/pending/<slug>/04-engineering.md` exists: scorecard table +
per-dimension evidence/gaps + top-gaps list.

## Stop Conditions

- A score without cited evidence is not a score — re-probe or mark the dimension
  unassessed.
- Route errata to earlier passes if probes invalidate their claims (e.g. a proposal
  name colliding with an existing CLI command); never silently rewrite their artifacts.

## Reference

- `deep-analysis-workflow.md` (master); inputs from Passes 0–3 where available.
- Worked example: `.analysis/pending/strategist-deep-analysis/04-engineering.md` (2026-07-26).
