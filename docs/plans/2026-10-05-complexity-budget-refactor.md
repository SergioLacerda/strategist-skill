# Complexity Budget Refactor Implementation Plan

> **For Codex:** Execute this plan in the current worktree while preserving unrelated staged and unstaged changes.

**Goal:** Reduce every reported production/test function to cognitive complexity `<= 7` without changing public APIs, observable behavior, error messages, or quality-budget configuration.

**Architecture:** Keep existing public functions as orchestration boundaries and extract private helpers around validation, classification, persistence, and rendering decisions. Use small typed helpers and explicit switch cases instead of changing control flow or introducing new abstractions. Refactor in package groups so each group can be tested independently.

**Tech Stack:** Go, `gocognit`, `golangci-lint`, package unit tests, isolated `GOCACHE` and `GOLANGCI_LINT_CACHE`.

---

### Task 1: Refactor Runbook preparation and comparison

**Files:**
- Modify: `internal/tools/prepare_runbook/compare.go`
- Modify: `internal/tools/prepare_runbook/prepare.go`
- Modify: `internal/tools/prepare_runbook/text_helpers.go`
- Test: `internal/tools/prepare_runbook/*_test.go`

Extract inventory validation/walking, sidecar loading, relation classification, frontmatter list conversion, and line matching helpers. Preserve comparison reasons, ordering, inventory limits, and provider invocation behavior.

Run: `GOCACHE=/tmp/strategist-go-cache go test ./internal/tools/prepare_runbook`

### Task 2: Refactor Leveling budget policy and ledger flows

**Files:**
- Modify: `internal/tools/leveling/merge.go`
- Modify: `internal/tools/leveling/budget_ledger.go`
- Modify: `internal/tools/leveling/budget_ledger_rotation.go`
- Modify: `internal/tools/leveling/budget_runtime.go`
- Modify: `internal/tools/leveling/policy_validate.go`
- Modify: `internal/tools/leveling/suggest.go`
- Test: `internal/tools/leveling/*_test.go`

Split policy precedence, override validation, ledger event decoding/encoding, rotation decisions, request-error construction, defaults validation, and suggestion classification into private helpers. Keep fail-closed outcomes and exact event/diagnostic formats.

Run: `GOCACHE=/tmp/strategist-go-cache go test ./internal/tools/leveling`

### Task 3: Refactor integration and Opportunity Attack flows

**Files:**
- Modify: `internal/integration/runbookconsumer/consumer.go`
- Modify: `internal/feats/opportunity_attack/evaluate.go`
- Modify: `internal/feats/opportunity_attack/persist.go`
- Modify: `cmd/strategist/mission/opportunity_attack_chest.go`
- Modify: `cmd/strategist/mission/opportunity_attack_prepare.go`
- Test: related package tests

Separate request projection, outcome decoding, chest-source filtering, manifest loading/persistence, and command orchestration. Preserve advisory-only provider semantics, digest checks, and no-network test boundaries.

Run: `GOCACHE=/tmp/strategist-go-cache go test ./internal/integration/runbookconsumer ./internal/feats/opportunity_attack ./cmd/strategist/mission`

### Task 4: Refactor mission, domain, install, missionview, and dojo flows

**Files:**
- Modify: `internal/domain/effort_budget.go`
- Modify: `internal/domain/effort_resolution_artifact.go`
- Modify: `internal/install/active_yaml.go`
- Modify: `internal/mission/side_quest_decide.go`
- Modify: `internal/missionview/render.go`
- Modify: `cmd/strategist/mission/view.go`
- Modify: `cmd/strategist/ranger_budget.go`
- Modify: `internal/dojo/calibration_check.go`
- Modify: `internal/feats/initiative/leveling_snapshot.go`
- Test: related package tests

Extract field validation, route/decision selection, confidence rendering, view loading, budget admission, calibration result assembly, and snapshot validation helpers. Preserve route precedence, authorization gates, and output formatting.

Run: `GOCACHE=/tmp/strategist-go-cache go test ./internal/domain ./internal/install ./internal/mission ./internal/missionview ./internal/dojo ./internal/feats/initiative ./cmd/strategist`

### Task 5: Verify the complete quality budget

Run:

```bash
GOCACHE=/tmp/strategist-go-cache /home/sergio/go/bin/gocognit -over 7 ./cmd ./internal ./treasure-chest
GOCACHE=/tmp/strategist-go-cache GOLANGCI_LINT_CACHE=/tmp/strategist-golangci-cache /home/sergio/go/bin/golangci-lint run ./...
git diff --check
```

Expected: no cognitive-complexity findings, `golangci-lint` reports `0 issues`, and the diff has no whitespace errors.
