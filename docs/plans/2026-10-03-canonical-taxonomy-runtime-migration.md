# Canonical Taxonomy Runtime Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate the embedded/runtime taxonomy from the former six-family vocabulary to ADR-0064's seven-family model without changing mission routing or authorization semantics.

**Architecture:** Keep `contracts/machine/mechanisms.yaml` as the compatibility registry path, but make its row families explicit for Mechanisms, Feats, and Tools. Reclassify `LEVELING` as a Tool and contextual behaviors such as `INITIATIVE` and Critical Hit as Feats; preserve routes as compatibility identifiers and preserve deterministic enforcement as Mechanisms. Rename the INITIATIVE runtime field and telemetry identifiers from Ability to Feat, then regenerate all sanctioned mirrors and generated contract indexes.

**Tech Stack:** Go, YAML contracts, embedded defaults, generated `.strategist/` runtime, deterministic Make targets, hermetic Go tests.

---

### Task 1: Establish the failing taxonomy contract tests

**Files:**
- Modify: `internal/mechanisms/registry_test.go`
- Modify: `internal/initiative/initiative_test.go`
- Modify: `internal/telemetry/*_test.go` where the current Ability telemetry key is asserted

**Step 1: Update test fixtures and expectations to the seven-family vocabulary**

Change registry fixtures and assertions to recognize `mechanism`, `feat`, and `tool`, and assert the canonical classifications for `leveling`, `initiative`, `critical_hit`, `opportunity_attack`, `riposte`, `search`, and `select_runbook`.

**Step 2: Add serialization/telemetry assertions for the Feat identity**

Assert that the INITIATIVE policy uses `feat: initiative` and telemetry emits `strategist.feat` plus the initiative Feat identity.

**Step 3: Run focused tests to capture the expected failures**

Run: `GOCACHE=/tmp/strategist-go-cache rtk go test ./internal/mechanisms ./internal/initiative ./internal/telemetry`

Expected: FAIL because the current registry, policy, and telemetry still expose `ability` and classify `LEVELING`/`INITIATIVE` under the former model.

### Task 2: Migrate the runtime taxonomy model and INITIATIVE identifiers

**Files:**
- Modify: `internal/mechanisms/registry.go`
- Modify: `internal/mechanisms/brief.go`
- Modify: `internal/initiative/policy_types.go`
- Modify: `internal/initiative/policy.go`
- Modify: `internal/mission/initiative_telemetry.go`
- Modify: `internal/telemetry/schema.go`
- Modify: `internal/embed/defaults/initiative.yaml`

**Step 1: Add explicit Feat and Tool family constants**

Extend the registry validator with `FamilyFeat` and `FamilyTool`; keep the registry filename and top-level YAML key stable as compatibility boundaries.

**Step 2: Rename INITIATIVE's public runtime field**

Replace `AbilityName`/`Policy.Ability` and the serialized `ability` field with `FeatName`/`Policy.Feat` and serialized `feat`, updating validation, default policy construction, parsing, and digest behavior.

**Step 3: Rename telemetry identifiers**

Emit `strategist.feat` and `strategist.initiative.feat` through `AttrFeat` and `AttrInitiativeFeat`; update the mission telemetry adapter and focused tests.

**Step 4: Run focused tests**

Run: `GOCACHE=/tmp/strategist-go-cache rtk go test ./internal/mechanisms ./internal/initiative ./internal/mission ./internal/telemetry`

Expected: PASS.

### Task 3: Reclassify embedded contracts and authoring taxonomy

**Files:**
- Modify: `internal/embed/defaults/contracts/machine/mechanisms.yaml`
- Modify: `internal/embed/defaults/skill.yaml`
- Modify: `internal/embed/defaults/SKILL.md`
- Modify: `internal/embed/defaults/contracts/narrative/10-telemetry.md`
- Modify: `internal/embed/defaults/schemas/initiative-advice.schema.yaml`
- Modify: `internal/embed/defaults/roles/{scout,ranger,archivist,sniper}.yaml`
- Modify: `internal/embed/defaults/potions.yaml`
- Modify: `internal/embed/defaults/schemas/handoff-ranger-to-archivist.schema.yaml`

**Step 1: Reclassify registry rows**

Use `tool` for `LEVELING` and operation-oriented rows, `feat` for `INITIATIVE`, Critical Hit, Opportunity Attack, Riposte, Search, runbook selection, and Keen Senses, and retain `mechanism` for deterministic invariants and enforcement.

**Step 2: Replace the authoring taxonomy prose**

Describe all seven families and explicitly state that routes are selection outcomes, not a family. Keep the existing runtime registry path as compatibility vocabulary.

**Step 3: Replace Ability terminology in contract prose and role briefs**

Use Feat consistently for contextual behavior while preserving unrelated uses of “capability” and the historical wording in ADRs.

**Step 4: Run the focused embedded and registry tests**

Run: `GOCACHE=/tmp/strategist-go-cache rtk go test ./internal/embed ./internal/mechanisms ./internal/check`

Expected: PASS after generated fixtures are synchronized.

### Task 4: Regenerate runtime and generated documentation artifacts

**Files:**
- Generated: `.strategist/`
- Generated: `docs/generated/`

**Step 1: Regenerate embedded defaults and runtime mirrors**

Run: `GOCACHE=/tmp/strategist-go-cache rtk make build-standalone` followed by
`GOCACHE=/tmp/strategist-go-cache rtk ./bin/strategist install --target /home/sergio/dev/strategist-skill --silent --strict-compile --no-shim`.

Expected: generated catalogs, locks, mirrors, and embedded digests update from `internal/embed/defaults/`.

**Step 2: Regenerate deterministic documentation indexes**

Run: `rtk make docs-generate`

Expected: generated contract/schema/event indexes are synchronized without unrelated changes.

**Step 3: Run drift checks**

Run: `GOCACHE=/tmp/strategist-go-cache rtk make embed-skills-check`; regenerate the docs twice and use
`rtk git diff --check` for the uncommitted implementation tree. `docs-generated-gate` compares against
`HEAD`, so it cannot pass until the intended generated diff is committed.

Expected: no embedded or generated documentation drift.

### Task 5: Verify behavior, compatibility boundaries, and quality

**Files:**
- Modify: `internal/mechanisms/registry_test.go` if any parity assertion needs tightening
- Modify: affected focused tests only when evidence identifies a stale identifier

**Step 1: Run targeted behavior tests**

Run: `GOCACHE=/tmp/strategist-go-cache rtk go test ./internal/mechanisms ./internal/initiative ./internal/mission ./internal/telemetry ./internal/embed ./internal/check`

Expected: PASS.

**Step 2: Verify the canonical/runtime boundary**

Run: `rtk rg -n -i "six public families|family: ability|strategist\.ability|ability: initiative|AbilityName|FamilyAbility" internal/embed/defaults .strategist internal/initiative internal/mechanisms internal/mission internal/telemetry`

Expected: no active runtime taxonomy identifiers remain; historical ADR references are outside this runtime search.

**Step 3: Run repository quality checks**

Run: `GOCACHE=/tmp/strategist-go-cache rtk make fmt-check quality-budget-gate`; then run
`rtk make docs-links-gate docs-index-ownership-gate contract-consistency-gate`.

Expected: PASS, with the unrelated pre-existing `.github/copilot-instructions.md` change preserved.

**Step 4: Review the final diff**

Run: `rtk git diff --check` and `rtk git status --short`.

Expected: only the taxonomy implementation, sanctioned generated artifacts, plan, and the already-existing unrelated worktree change are present.

Implementation evidence: focused tests passed (823 assertions), repository tests passed (4692 assertions),
embedded drift passed, quality budgets passed, documentation links/index ownership/contract consistency passed,
and the active runtime search found no legacy taxonomy identifiers.
