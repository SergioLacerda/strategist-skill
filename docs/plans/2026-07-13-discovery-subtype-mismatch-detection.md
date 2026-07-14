# Discovery Subtype/Weapon Mismatch: Early Detection and Honest Messaging Implementation Plan

> **REQUIRED SUB-SKILL:** Use executing-plans to implement this plan task-by-task.

**Goal:** Move `provider_capability_mismatch` detection to run immediately after Scout emits `route_decision.discovery_subtype` (before the discovery weapon is invoked), and correct the remediation hint so it doesn't imply editing `skill.yaml` metadata alone fixes an external weapon's real behavior.

**Architecture:** This is a contract-text-only change (narrative markdown + machine YAML), following the exact same pattern as the rest of the Scout feature (T2–T9): edit `strategist/**`, mirror byte-identically into `internal/embed/defaults/**`, update the two spec-alignment tests that assert this content. No Go code changes — Scout, Ranger, and `provider_capability_mismatch` as a failure code are prompt-time/LLM-runtime concepts, not compiled logic.

**Tech Stack:** Markdown/YAML contract files, Go test assertions in `tests/spec/spec_alignment_test.go` (build tag `spec`), `make sync-embed` for tree parity.

---

## Before You Start

Read the approved design doc first: `.analysis/pending/2026-07-13-discovery-subtype-mismatch-detection-design.md`. It explains why this is narrow in scope (no fallback/bypass mechanism, no per-subtype weapon configuration — those were considered and explicitly rejected).

Read `strategist/contracts/narrative/00-routing.md` § "Scout — Intake Router" (lines 38–64) and `strategist/contracts/machine/scout-routing.yaml` in full before editing — you need to match their existing prose/YAML style exactly.

Every edit below touches `strategist/<path>` (canonical source). After each file's edit, the exact same content must be copied byte-for-byte into `internal/embed/defaults/<path>`. Use `diff -q` to confirm — never eyeball it.

---

### Task 1: Add the post-route capability check to `00-routing.md`

**Files:**
- Modify: `strategist/contracts/narrative/00-routing.md` (insert after line 64, right before `## Main Mission Sequence`)
- Mirror: `internal/embed/defaults/contracts/narrative/00-routing.md`
- Test: `tests/spec/spec_alignment_test.go` (new test, see Task 5)

**Step 1: Write the failing test first**

Add this test to `tests/spec/spec_alignment_test.go` at the end of the file (after the last existing test):

```go
// TestRoutingContractDefinesPostRouteCapabilityCheck verifies 00-routing.md
// describes the weapon-capability check running immediately after Scout emits
// route_decision, before the discovery weapon is invoked — not at classic
// preflight time (which runs before Scout exists) and not discovered empirically
// by invoking the weapon.
func TestRoutingContractDefinesPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "strategist", "contracts", "narrative", "00-routing.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"post_route_capability_check",
			"before the discovery weapon is invoked",
			"discovery_subtype_support",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing post-route capability check term %q", path, needle)
			}
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/spec/... -tags spec -run TestRoutingContractDefinesPostRouteCapabilityCheck -v`

Expected: FAIL — `strategist/contracts/narrative/00-routing.md missing post-route capability check term "post_route_capability_check"` (the text doesn't exist yet).

**Step 3: Write the contract text**

In `strategist/contracts/narrative/00-routing.md`, insert this new section right after the existing Scout paragraph (after the line ending `...and must select \`full_pipeline\` instead.` and before `## Main Mission Sequence`):

```markdown
### Post-Route Capability Check

Immediately after Scout emits `route_decision` with `evidence_state: requires_discovery`
(before the discovery weapon is invoked), check the resolved weapon's `skill.yaml`
`discovery_subtype_support` field against the required `discovery_subtype`. This runs
as part of Scout's routing responsibility (see `contracts/machine/scout-routing.yaml`
§ `post_route_capability_check`) — not at classic preflight time, since preflight
runs before intake/routing and before `discovery_subtype` exists.

If the resolved weapon does not declare support for the required `discovery_subtype`,
emit `provider_capability_mismatch` and stop **before** invoking the weapon. Do not
invoke the weapon to discover the mismatch empirically — that wastes an invocation and
risks the weapon partially acting before the mismatch is caught. See `preflight.yaml`
for the full error condition and remediation hint.
```

**Step 4: Run test to verify it passes**

Run: `go test ./tests/spec/... -tags spec -run TestRoutingContractDefinesPostRouteCapabilityCheck -v`

Expected: PASS

**Step 5: Mirror into embedded defaults**

```bash
diff -u strategist/contracts/narrative/00-routing.md internal/embed/defaults/contracts/narrative/00-routing.md
```

If this shows a diff (it will, since you only edited the source copy), copy it over:

```bash
cp strategist/contracts/narrative/00-routing.md internal/embed/defaults/contracts/narrative/00-routing.md
diff -q strategist/contracts/narrative/00-routing.md internal/embed/defaults/contracts/narrative/00-routing.md
```

The second command must produce no output (files identical) — if it prints anything, the mirror failed; re-copy.

**Step 6: Run the test again against both trees**

Run: `go test ./tests/spec/... -tags spec -run TestRoutingContractDefinesPostRouteCapabilityCheck -v`

Expected: PASS (both source and embedded-defaults subtests pass)

**Step 7: Commit**

```bash
git add strategist/contracts/narrative/00-routing.md internal/embed/defaults/contracts/narrative/00-routing.md tests/spec/spec_alignment_test.go
git commit -m "docs: describe post-route capability check in routing contract"
```

---

### Task 2: Add `post_route_capability_check` block to `scout-routing.yaml`

**Files:**
- Modify: `strategist/contracts/machine/scout-routing.yaml` (append near the end, after the `invariants:` block)
- Mirror: `internal/embed/defaults/contracts/machine/scout-routing.yaml`
- Test: already written in Task 1 (`TestRoutingContractDefinesPostRouteCapabilityCheck` doesn't check this file directly — write a dedicated test here)

**Step 1: Write the failing test**

Add to `tests/spec/spec_alignment_test.go`, right after `TestRoutingContractDefinesPostRouteCapabilityCheck`:

```go
// TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck verifies
// scout-routing.yaml defines the post_route_capability_check block that runs
// before weapon invocation, checking discovery_subtype_support.
func TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "strategist", "contracts", "machine", "scout-routing.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "scout-routing.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"post_route_capability_check:",
			"discovery_subtype_support",
			"before_weapon_invocation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing post_route_capability_check term %q", path, needle)
			}
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/spec/... -tags spec -run TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck -v`

Expected: FAIL

**Step 3: Write the YAML block**

Append this to `strategist/contracts/machine/scout-routing.yaml`, after the closing line of the existing `invariants:` list (`- gate_required is always true, regardless of selected_route`):

```yaml

post_route_capability_check:
  description: >
    Runs immediately after route_decision is emitted with
    evidence_state: requires_discovery, and before the resolved discovery weapon is
    invoked. This is Scout's routing responsibility, not classic preflight — preflight
    runs before intake/routing, before discovery_subtype exists.
  timing: before_weapon_invocation
  check: >
    Read the resolved discovery weapon's skill.yaml discovery_subtype_support field.
    If it does not declare support (native or adapter) for route_decision.discovery_subtype,
    do not invoke the weapon. Emit provider_capability_mismatch instead (see
    contracts/machine/preflight.yaml for the full error condition and remediation hint).
  emit:
    on_mismatch:
      format: >
        "[Strategist] phase=intake status=blocked reason=provider_capability_mismatch
         slot=discovery weapon={weapon_id} subtype={discovery_subtype}"
      level: INFO
```

**Step 4: Run test to verify it passes**

Run: `go test ./tests/spec/... -tags spec -run TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck -v`

Expected: PASS (source copy only — embedded-defaults subtest still fails, that's expected until Step 5)

**Step 5: Mirror into embedded defaults**

```bash
cp strategist/contracts/machine/scout-routing.yaml internal/embed/defaults/contracts/machine/scout-routing.yaml
diff -q strategist/contracts/machine/scout-routing.yaml internal/embed/defaults/contracts/machine/scout-routing.yaml
```

Second command must produce no output.

**Step 6: Run the test again**

Run: `go test ./tests/spec/... -tags spec -run TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck -v`

Expected: PASS (both subtests)

**Step 7: Commit**

```bash
git add strategist/contracts/machine/scout-routing.yaml internal/embed/defaults/contracts/machine/scout-routing.yaml tests/spec/spec_alignment_test.go
git commit -m "docs: add post_route_capability_check block to scout-routing contract"
```

---

### Task 3: Correct the `preflight.yaml` trigger and hint

**Files:**
- Modify: `strategist/contracts/machine/preflight.yaml:54-66`
- Mirror: `internal/embed/defaults/contracts/machine/preflight.yaml`
- Test: Modify existing `TestPreflightContractDefinesProviderCapabilityMismatch` (`tests/spec/spec_alignment_test.go:2239`)

**Step 1: Update the test's assertions first (TDD — this makes the test fail against current content)**

In `tests/spec/spec_alignment_test.go`, find `TestPreflightContractDefinesProviderCapabilityMismatch` (currently at line 2239) and replace its needle list:

```go
func TestPreflightContractDefinesProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "strategist", "contracts", "machine", "preflight.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "preflight.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"code: provider_capability_mismatch",
			"reason=provider_capability_mismatch",
			"scout-routing.yaml",
			"editing skill.yaml metadata alone does not change",
			"Verify with a live invocation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing preflight provider_capability_mismatch term %q", path, needle)
			}
		}
	}
}
```

Note: this removes the old `"Do not fall back to parent-agent discovery"` assertion — that phrase stays true conceptually but isn't the focus of this fix; the new needles target the corrected hint and the cross-reference to `scout-routing.yaml`.

**Step 2: Run test to verify it fails**

Run: `go test ./tests/spec/... -tags spec -run TestPreflightContractDefinesProviderCapabilityMismatch -v`

Expected: FAIL — missing `"scout-routing.yaml"` and the new hint text (current file doesn't have them yet).

**Step 3: Edit `preflight.yaml`**

In `strategist/contracts/machine/preflight.yaml`, replace lines 54–66 (the entire `provider_capability_mismatch` block) with:

```yaml
    - code: provider_capability_mismatch
      trigger: >
        The resolved discovery weapon is invocable but does not declare support
        (native or adapter) for the required discovery_subtype. Checked by Scout's
        post_route_capability_check (see contracts/narrative/00-routing.md § Post-Route
        Capability Check and contracts/machine/scout-routing.yaml
        § post_route_capability_check) immediately after route_decision is emitted,
        before the weapon is invoked — not at this preflight phase, which runs before
        discovery_subtype exists.
      behavior: >
        Emit:
          "[Strategist] phase=preflight status=blocked reason=provider_capability_mismatch
           slot=discovery provider=<provider_id> subtype=<discovery_subtype>"
        Stop mission. Do not fall back to parent-agent discovery.
        Hint: configure active.slots.discovery to a weapon whose own instructions
        actually implement this discovery_subtype — editing skill.yaml metadata alone
        does not change an external weapon's real behavior. Verify with a live
        invocation before declaring discovery_subtype_support: adapter for any subtype.
```

**Step 4: Run test to verify it passes**

Run: `go test ./tests/spec/... -tags spec -run TestPreflightContractDefinesProviderCapabilityMismatch -v`

Expected: PASS (source copy — embedded-defaults subtest still fails until Step 5)

**Step 5: Mirror into embedded defaults**

```bash
cp strategist/contracts/machine/preflight.yaml internal/embed/defaults/contracts/machine/preflight.yaml
diff -q strategist/contracts/machine/preflight.yaml internal/embed/defaults/contracts/machine/preflight.yaml
```

**Step 6: Run test again, plus the normative parity regression test**

Run: `go test ./tests/spec/... -tags spec -run 'TestPreflightContractDefinesProviderCapabilityMismatch|TestNormativeRuntimeFilesMirrorEmbeddedDefaults' -v`

Expected: PASS. `preflight.yaml` is in the normative byte-identical list (`normativeRuntimeFiles()`), so `TestNormativeRuntimeFilesMirrorEmbeddedDefaults` must also pass — this confirms the mirror is exact.

**Step 7: Commit**

```bash
git add strategist/contracts/machine/preflight.yaml internal/embed/defaults/contracts/machine/preflight.yaml tests/spec/spec_alignment_test.go
git commit -m "fix: correct provider_capability_mismatch trigger timing and remediation hint"
```

---

### Task 4: Correct the `drift-patterns.yaml` correction text (normative file)

**Files:**
- Modify: `strategist/templates/domain/identity/drift-patterns.yaml:28-40`
- Mirror: `internal/embed/defaults/templates/domain/identity/drift-patterns.yaml`
- Also refresh: `.strategist/templates/domain/identity/drift-patterns.yaml` (this workspace's runtime mirror — see note below)
- Test: Modify existing `TestDriftPatternsCoverProviderCapabilityMismatch` (`tests/spec/spec_alignment_test.go:2261`)

This file is in the **normative** byte-identical-enforced list — treat the mirror step as non-optional and always verify with `diff -q`.

**Step 1: Update the test's assertions first**

Replace `TestDriftPatternsCoverProviderCapabilityMismatch`:

```go
func TestDriftPatternsCoverProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "strategist", "templates", "domain", "identity", "drift-patterns.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "domain", "identity", "drift-patterns.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"id: provider_capability_mismatch",
			"editing skill.yaml metadata alone does not change",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing provider_capability_mismatch drift pattern term %q", path, needle)
			}
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/spec/... -tags spec -run TestDriftPatternsCoverProviderCapabilityMismatch -v`

Expected: FAIL — missing the corrected hint phrase.

**Step 3: Edit `drift-patterns.yaml`**

In `strategist/templates/domain/identity/drift-patterns.yaml`, replace lines 28–40 (the `provider_capability_mismatch` pattern) with:

```yaml
  - id: provider_capability_mismatch
    symptom: >
      Scout selected discovery_subtype=evaluation (or another non-creative subtype),
      and I am about to invoke the configured discovery weapon to find out whether it
      supports this subtype — instead of checking discovery_subtype_support first.
    correction: >
      Stop before invoking the weapon. Check the resolved weapon's skill.yaml
      discovery_subtype_support field against route_decision.discovery_subtype (see
      contracts/machine/scout-routing.yaml § post_route_capability_check). If
      unsupported, emit blocked reason=provider_capability_mismatch with slot=discovery,
      the configured weapon, and the required discovery_subtype. Never fall back to
      parent-agent discovery. Instruct the user: configure active.slots.discovery to a
      weapon whose own instructions actually implement this discovery_subtype — editing
      skill.yaml metadata alone does not change an external weapon's real behavior.
```

**Step 4: Run test to verify it passes**

Run: `go test ./tests/spec/... -tags spec -run TestDriftPatternsCoverProviderCapabilityMismatch -v`

Expected: PASS (source copy only)

**Step 5: Mirror into embedded defaults**

```bash
cp strategist/templates/domain/identity/drift-patterns.yaml internal/embed/defaults/templates/domain/identity/drift-patterns.yaml
diff -q strategist/templates/domain/identity/drift-patterns.yaml internal/embed/defaults/templates/domain/identity/drift-patterns.yaml
```

**Step 6: Refresh this workspace's `.strategist/` runtime copy**

This workspace has its own `.strategist/templates/domain/identity/drift-patterns.yaml` installed copy. Refresh it directly (do NOT run `strategist install --force` — that would overwrite `active.yaml`, see the T2-T9 session notes if unsure why):

```bash
cp strategist/templates/domain/identity/drift-patterns.yaml .strategist/templates/domain/identity/drift-patterns.yaml
diff -q strategist/templates/domain/identity/drift-patterns.yaml .strategist/templates/domain/identity/drift-patterns.yaml
```

Also refresh the other three files this plan touched, the same way:

```bash
cp strategist/contracts/narrative/00-routing.md .strategist/contracts/narrative/00-routing.md
cp strategist/contracts/machine/scout-routing.yaml .strategist/contracts/machine/scout-routing.yaml
cp strategist/contracts/machine/preflight.yaml .strategist/contracts/machine/preflight.yaml
for f in contracts/narrative/00-routing.md contracts/machine/scout-routing.yaml contracts/machine/preflight.yaml templates/domain/identity/drift-patterns.yaml; do
  diff -q "strategist/$f" ".strategist/$f"
done
```

All four `diff -q` calls must produce no output.

**Step 7: Run full spec suite and the normative parity test**

Run: `go test ./tests/spec/... -tags spec -v 2>&1 | tail -60`

Expected: PASS across the board, including `TestNormativeRuntimeFilesMirrorEmbeddedDefaults` and `TestLocalRuntimeMirrorsCanonicalNormativeFilesWhenPresent`.

**Step 8: Commit**

```bash
git add strategist/templates/domain/identity/drift-patterns.yaml internal/embed/defaults/templates/domain/identity/drift-patterns.yaml tests/spec/spec_alignment_test.go
git commit -m "fix: correct provider_capability_mismatch drift-pattern remediation hint"
```

Note: `.strategist/` is typically gitignored (runtime install artifact) — check `git status` before staging; if it's tracked in this repo for some reason, include it in the commit too, otherwise leave it untracked/local-only.

---

### Task 5: Full regression pass and rebuild

**Files:** none (verification only)

**Step 1: Run the full spec suite**

Run: `go test ./tests/spec/... -tags spec -v 2>&1 | tail -100`

Expected: every test PASSes, including all four new/modified tests from Tasks 1–4 and every pre-existing test (especially `TestNormativeRuntimeFilesMirrorEmbeddedDefaults`, `TestLocalRuntimeMirrorsCanonicalNormativeFilesWhenPresent`, `TestRoutingContractExcludesCodeMutationFromCriticalHit`, and everything from the earlier Scout T2–T9 work).

**Step 2: Run the full Go test suite (no build tag)**

Run: `go build ./... && go test ./...`

Expected: builds cleanly, all packages PASS. This plan makes no Go code changes, so this should be unaffected — if anything fails here, it's unrelated to this plan (check `git status` for other in-flight work before investigating).

**Step 3: Run `make sync-embed` as a final safety net**

Run: `make sync-embed`

Expected: no unexpected diffs beyond what Tasks 1–4 already produced. Run `git status --short` afterward and confirm only the files this plan touched show as modified (plus any pre-existing unrelated in-flight work already in the tree — do not touch those).

**Step 4: Rebuild the binary and refresh this workspace's compiled runtime**

```bash
go build -o bin/strategist ./cmd/strategist
./bin/strategist compile --root .strategist
./bin/strategist check
```

Expected: `strategist check` reports `STATUS ok`.

**Step 5: Final commit (if anything remains uncommitted)**

```bash
git status --short
```

If `bin/strategist` or `.strategist/.compiled/` show as modified, these are build artifacts — check whether they're gitignored before deciding to commit them; do not force-add ignored files.

---

## Explicitly Out of Scope (do not implement)

- Any fallback/bypass mechanism where Ranger acts "unarmed" — rejected during design; Ranger always wields a weapon.
- Per-subtype configurable weapon selection in `active.yaml` (e.g. `discovery_weapons: {creative: ..., evaluation: ...}`) — not requested.
- Any change to Scout's core classification logic, the `discovery_subtype` vocabulary, or the `evaluation_verdict` mechanism from T2–T9 — those stand as implemented.
- Any Go code change — this is contract-text only.
