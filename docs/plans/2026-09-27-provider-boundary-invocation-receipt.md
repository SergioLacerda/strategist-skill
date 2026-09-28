# Provider Boundary Invocation Receipt Implementation Plan

**Goal:** Make delegated Ranger invocation attributable, pin-verifiable, and fail-closed.

**Architecture:** Extend the connector result with a host-issued receipt; validate it at the provider normalization boundary before accepting an artifact. Reuse catalog pin comparison and emit bounded telemetry, while keeping capability isolation explicitly unverified unless a host proves it.

**Tech Stack:** Go, Cobra, YAML, testify.

---

### Task 1: Receipt model and connector contract

**Files:** Modify `internal/plugins/connectors/*`; Test `internal/plugins/connectors/*_test.go`.

1. Write failing tests for required mission, role, Weapon, digest, timestamp, nonce and isolation state.
2. Add the receipt type and validate malformed/empty fields without raw payloads.
3. Extend `ConnectorResult`/`HostWeaponConnector` to carry only the receipt.
4. Run `go test ./internal/plugins/connectors`.

### Task 2: Ranger fail-closed verification

**Files:** Modify `internal/provider/discovery_connector.go`, `internal/provider/discovery_normalize.go`; Test `internal/provider/discovery_connector_test.go`.

1. Write tests for absent, mismatched provider/mission/role, stale, and replayed receipts.
2. Validate the receipt before normalization; wrap all rejection paths as `role_invocation_failed`.
3. Compare a declared catalog pin using `internal/install.CompareResolvedDigest`; report unpinned distinctly.
4. Run `go test ./internal/provider ./internal/mission`.

### Task 3: Telemetry and host conformance

**Files:** Modify `internal/telemetry/discovery.go`; Test `internal/telemetry/discovery_test.go`.

1. Add redacted receipt outcome attributes for authenticated invocation, pin state, and capability-isolation state.
2. Add Codex/Claude conformance fixtures: valid receipt, unavailable receipt, mismatch, replay, and `unverified` isolation.
3. Run focused telemetry and connector tests.

### Task 4: Canonical contract and parity

**Files:** Modify `internal/embed/defaults/contracts/narrative/03-discovery.md`, schemas, catalog authority docs; Test source/runtime parity.

1. Document receipt semantics and prohibit claiming isolation from receipt alone.
2. Regenerate `.strategist/` from canonical embedded source.
3. Run focused tests, `strategist check --json`, and parity validation.

No Git commit is included: M010 requires an explicit separate request.
