# Client conformance

Client conformance uses the matrix fixture at
`tests/conformance/testdata/client-matrix.yaml` and the reusable validator in
`internal/conformance`.

The current release-supported structural inventory is:

| Client surface | Owner | Evidence boundary |
|---|---|---|
| `.codex` | Codex adapter | Structural envelope and explicit probe state |
| `.claude` | Claude adapter | Structural envelope and explicit probe state |
| `.gemini/antigravity` | Gemini adapter | Structural envelope and explicit probe state |
| `internal/governance` | Governance adapter | Structural envelope and explicit probe state |

Each matrix row declares client, role, slot, provider mode, envelope version,
evidence tier, expected state, reason code, authority owner, and context fixture.
Adding a supported client requires an inventory row, at least one matrix row,
an owner, and an unsupported-provider policy. Unclassified clients fail matrix
validation.

Structural rows are hermetic and do not invoke a provider. They validate shared
identity, authority, digest, ordering, bounds, provenance, and fail-closed
reason semantics. Live rows remain pending until a successful ready probe is
provided; static catalog or manifest metadata never substitutes for that probe.

Evidence dimensions remain separate:

- static evidence describes checked-in contracts and manifests;
- persisted evidence describes pinned identity, digest, provenance, and release
  records;
- structural evidence is produced by the local matrix without provider
  invocation;
- live evidence is a redacted `strategist-live-evidence/v1` envelope tied to a
  provider, authorized runner, matrix row, bounded timeout, teardown result,
  report location, and retention policy;
- manual Promptfoo evidence is optional operator-collected evidence and never
  certifies an automated live row by itself;
- remote governance and published-release evidence require their own
  authorized remote source.

The live executor requires an authorization reference, timeout, teardown, and
report/retention metadata. It records `unavailable`, `unauthorized`,
`timeout`, `malformed`, `failed`, or `teardown_failed` as non-certified states;
it never serializes credentials or provider payloads. No provider or runner is
selected by the local contract.

The coverage inventory is checked independently by
`make coverage-manifest-check`. It discovers `cmd/...`, `internal/...`, and
`treasure-chest/...` with Go tooling and compares the result with
`scripts/coverage-packages.tsv` plus the owner/reason records in
`scripts/coverage-exemptions.tsv`.

For optional Promptfoo evidence, first run the guarded endpoint preflight and
then collect the report manually:

```bash
make eval-promptfoo PROMPTFOO_LM_STUDIO_URL=http://127.0.0.1:1234/v1
```

The command remains outside default CI. A failed preflight is non-evidence and
must not be converted into a passing live or structural result.

Run the focused suite with isolated caches:

```bash
GOCACHE=/tmp/go-cache-phase-c GOMODCACHE=/home/sergio/go/pkg/mod \
  go test ./internal/conformance ./tests/conformance \
    ./internal/plugins/conformance ./internal/plugins/lifecycle \
    ./internal/plugins/connectors
```

The matrix report is deterministic for a fixed fixture and availability map.
Disabling a future cross-client gate must not disable the existing component
tests or Ranked/Custom fail-closed checks.
