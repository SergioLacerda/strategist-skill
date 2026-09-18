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
