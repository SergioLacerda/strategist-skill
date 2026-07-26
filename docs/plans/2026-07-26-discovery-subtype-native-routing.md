# Discovery Subtype Native Routing Implementation Plan

> **REQUIRED SUB-SKILL:** Use executing-plans to implement this plan task-by-task.

**Goal:** Make `discovery_subtype: evaluation | diagnostic | closure_evidence` requests always route to the native `internal_skills/ranger` role instead of the external weapon configured at `active.slots.discovery` (e.g. `brainstorming`), which today is invoked for every subtype regardless of whether its own instructions can actually honor Strategist's role/subtype contract.

**Architecture:** All changes land in `internal/embed/defaults/` — the single `go:embed` authoring source for `.strategist/`. No `.strategist/*` file is edited directly anywhere (that's a generated artifact; see design doc). Discovery invocation becomes conditional on `discovery_subtype`: `creative` keeps going to `{{.Slots.Discovery}}` (`kind=skill_provider`, invoked via the `Skill` tool); `evaluation`/`diagnostic`/`closure_evidence` always resolve to `internal_skills/ranger` (`kind=native_role`, the parent agent embodies Ranger directly by reading `roles/ranger.yaml` + `internal_skills/ranger/SKILL.md`) — mirroring the native-role mechanism already used for execution/`sniper`. The false `evaluation: adapter` / `diagnostic: adapter` / `closure_evidence: adapter` claims are removed from `brainstorming`'s manifest since that weapon is never consulted for those subtypes anymore, and a pre-existing test fixture (`provider_manifest_is_slot_authority` / `brainstorming_diagnostic_not_blocked_by_standalone_creative_first`) that encoded the old, now-falsified assumption ("trust the manifest without checking live behavior") is retired and replaced.

**Tech Stack:** Go 1.26, `go test -tags=spec ./tests/spec/...` (corpus/spec-alignment tests over `internal/embed/defaults/`), plain-text Markdown/YAML contract authoring (no template engine changes needed — `internal/compile/agent_protocol.go` already renders `{{.Slots.*}}` verbatim; this plan only changes the rendered template's prose/rule text, not the Go renderer).

**Governance note (this execution):** the user who approved this plan explicitly asked that no git mutation commands (`git add`, `git commit`, etc.) be run automatically. Each task below ends with a "Commit" step per the standard plan template — these are included as documentation of the intended commit boundary for whoever executes this plan, but must NOT be run without a separate, explicit go-ahead at execution time.

---

### Task 1: Conditional discovery routing in the compiled agent-protocol template

**Files:**
- Modify: `internal/embed/defaults/templates/agent-protocol.md:65-79` (§3 ROLE INVOCATION MODEL), `:92` (§4 item 5), `:45` (§2 forbidden behaviors line)
- Test: `tests/spec/spec_alignment_test.go` (new test, append near other agent-protocol-template tests — e.g. after `TestDiscoveryContractDefinesSubtypeVocabulary` around line 2075)

**Step 1: Write the failing test**

Add to `tests/spec/spec_alignment_test.go`:

```go
// TestAgentProtocolTemplateRoutesDiscoveryBySubtype verifies the compiled
// agent-protocol template resolves discovery invocation conditionally on
// discovery_subtype instead of unconditionally naming {{.Slots.Discovery}}.
func TestAgentProtocolTemplateRoutesDiscoveryBySubtype(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "agent-protocol.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"Discovery Routing",
		"discovery_subtype",
		"internal_skills/ranger",
		"native_role",
		"regardless of what `active.slots.discovery` is configured to",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing discovery-by-subtype routing term %q", path, needle)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestAgentProtocolTemplateRoutesDiscoveryBySubtype ./tests/spec/... -v`
Expected: FAIL — current template has none of these terms (it only has `{{.Slots.Discovery}}` unconditionally).

**Step 3: Edit the template**

In `internal/embed/defaults/templates/agent-protocol.md`, replace lines 65-79:

```markdown
## 3. ROLE INVOCATION MODEL

The providers below are read from `.strategist/active.yaml` at compile time. If `active.yaml` changes, run `strategist compile` to update this file.

```
PHASE         INVOKE SKILL                              WHAT NOT TO DO
─────────────────────────────────────────────────────────────────────────────
discovery  →  see Discovery Routing below                explore or analyze the code directly
refinement →  {{.Slots.Refinement}}                       write proposals or designs directly
execution  →  {{.Slots.Execution}}                        run git/edits/commits directly
```

### Discovery Routing

Discovery invocation target depends on `route_decision.discovery_subtype`
(set by Scout — see `00-routing.md` § Scout — Intake Router and
§ Discovery Weapon Resolution by Subtype):

| `discovery_subtype` | Invoke | Kind |
|---|---|---|
| `creative` | `{{.Slots.Discovery}}` | `skill_provider` — external weapon, invoked via the `Skill` tool |
| `evaluation` \| `diagnostic` \| `closure_evidence` | `internal_skills/ranger` | `native_role` — parent agent embodies Ranger directly (same mechanism already used for execution/`sniper`), reading `roles/ranger.yaml` + `internal_skills/ranger/SKILL.md` |

For `evaluation`/`diagnostic`/`closure_evidence` this holds regardless of what
`active.slots.discovery` is configured to — the external weapon is never
consulted for these subtypes. See `03-discovery.md` § Discovery Subtypes and
`contracts/machine/scout-routing.yaml` § `post_route_capability_check`.

Handoff contracts:
- Ranger → Archivist: `.strategist/schemas/handoff-ranger-to-archivist.schema.yaml`
- Archivist → Sniper: `.strategist/schemas/handoff-archivist-to-sniper.schema.yaml`
```

Then update line 92 (§4 PIPELINE SEQUENCE item 5), from:

```
[ ] 5. discovery → invoke {{.Slots.Discovery}}
```

to:

```
[ ] 5. discovery → invoke per Discovery Routing (§3): {{.Slots.Discovery}} for creative, internal_skills/ranger otherwise
```

Then update line 45 (§2 FORBIDDEN BEHAVIORS), from:

```
- Never invoke a discovery weapon when its manifest lacks `discovery_subtype_support` for Scout's required subtype — stop with `error=provider_capability_mismatch`
```

to:

```
- Never invoke a discovery weapon when its manifest lacks `discovery_subtype_support` for the required subtype — stop with `error=provider_capability_mismatch`. This check only ever applies to `creative`: `evaluation`/`diagnostic`/`closure_evidence` resolve directly to `internal_skills/ranger` and never consult the weapon's manifest (see §3 Discovery Routing).
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestAgentProtocolTemplateRoutesDiscoveryBySubtype ./tests/spec/... -v`
Expected: PASS

Also re-run the existing renderer test to confirm the template still parses as valid Go template syntax:
Run: `GOCACHE=/tmp/go-build-cache go test ./internal/compile/... -run TestAgentProtocol -v`
Expected: PASS (this test uses its own minimal inline template, not the production one, so it should be unaffected — it just guards that `agentProtocol()` itself still works)

**Step 5: Commit**

```bash
git add internal/embed/defaults/templates/agent-protocol.md tests/spec/spec_alignment_test.go
git commit -m "feat(discovery): route non-creative subtypes to native Ranger role in agent-protocol template"
```

---

### Task 2: Normative routing rule in 00-routing.md

**Files:**
- Modify: `internal/embed/defaults/contracts/narrative/00-routing.md` (insert new section after line 79, before line 81 `## Main Mission Sequence`)
- Test: `tests/spec/spec_alignment_test.go` (new test)

**Step 1: Write the failing test**

```go
// TestRoutingContractDefinesDiscoveryWeaponResolutionBySubtype verifies
// 00-routing.md normatively states that evaluation/diagnostic/closure_evidence
// discovery subtypes always resolve to internal_skills/ranger, bypassing the
// configured external weapon.
func TestRoutingContractDefinesDiscoveryWeaponResolutionBySubtype(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"Discovery Weapon Resolution by Subtype",
		"internal_skills/ranger",
		"kind=native_role",
		"regardless of `active.slots.discovery`",
		"never a live behavior guarantee",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing discovery weapon resolution term %q", path, needle)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestRoutingContractDefinesDiscoveryWeaponResolutionBySubtype ./tests/spec/... -v`
Expected: FAIL

**Step 3: Edit the contract**

In `internal/embed/defaults/contracts/narrative/00-routing.md`, insert this new subsection immediately after the existing `### Post-Route Capability Check` section (after line 79, before `## Main Mission Sequence`):

```markdown
### Discovery Weapon Resolution by Subtype

Discovery invocation target depends on `discovery_subtype`, not solely on
`active.slots.discovery`:

- `creative` → the external weapon configured at `active.slots.discovery`
  (`kind=skill_provider`), subject to the Post-Route Capability Check above.
- `evaluation` | `diagnostic` | `closure_evidence` → always
  `internal_skills/ranger` (`kind=native_role`), regardless of
  `active.slots.discovery`. The parent agent embodies Ranger directly — the
  same native-role mechanism already used for execution/`sniper` — reading
  `roles/ranger.yaml` + `internal_skills/ranger/SKILL.md` and performing
  discovery under that contract. The Post-Route Capability Check above does
  not run for these three subtypes: the external weapon's manifest is never
  consulted, because the weapon is never a candidate for these subtypes.

This exists because an external weapon's own `SKILL.md` is authored
independently of Strategist and cannot be relied on to honor
`roles/ranger.yaml` or subtype-specific obligations, even when its manifest
declares `adapter` support — declared support in a manifest is a capability
claim by whoever wrote it, never a live behavior guarantee. Only
`internal_skills/ranger`, authored by Strategist itself, can be trusted to
compose with `roles/ranger.yaml` per its own documented "Invocation Contract".
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestRoutingContractDefinesDiscoveryWeaponResolutionBySubtype ./tests/spec/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/embed/defaults/contracts/narrative/00-routing.md tests/spec/spec_alignment_test.go
git commit -m "docs(routing): normatively define discovery weapon resolution by subtype"
```

---

### Task 3: Cross-reference in 03-discovery.md

**Files:**
- Modify: `internal/embed/defaults/contracts/narrative/03-discovery.md` (insert paragraph after line 25, before line 27 `## Inputs`)
- Test: `tests/spec/spec_alignment_test.go` (new test)

**Step 1: Write the failing test**

```go
// TestDiscoveryContractCrossReferencesSubtypeResolution verifies 03-discovery.md
// points to 00-routing.md for which concrete invocation target (external weapon
// vs. native Ranger) handles each subtype.
func TestDiscoveryContractCrossReferencesSubtypeResolution(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "03-discovery.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"Discovery Weapon Resolution by Subtype",
		"internal_skills/ranger",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing subtype resolution cross-reference term %q", path, needle)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestDiscoveryContractCrossReferencesSubtypeResolution ./tests/spec/... -v`
Expected: FAIL

**Step 3: Edit the contract**

In `internal/embed/defaults/contracts/narrative/03-discovery.md`, insert after line 25 (end of the subtype table) and before line 27 (`## Inputs`):

```markdown

Resolution of which concrete invocation target handles a given subtype (the
configured external weapon vs. the native `internal_skills/ranger` role) is
defined in `00-routing.md` § Discovery Weapon Resolution by Subtype — Ranger's
own behavior below is identical regardless of which mechanism invoked it.
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestDiscoveryContractCrossReferencesSubtypeResolution ./tests/spec/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/embed/defaults/contracts/narrative/03-discovery.md tests/spec/spec_alignment_test.go
git commit -m "docs(discovery): cross-reference subtype weapon resolution in 03-discovery.md"
```

---

### Task 4: Scope `post_route_capability_check` to `creative` in scout-routing.yaml

**Files:**
- Modify: `internal/embed/defaults/contracts/machine/scout-routing.yaml:80-97`
- Test: `tests/spec/spec_alignment_test.go` (modify existing `TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck`, currently at lines 2351-2368)

**Step 1: Write the failing test**

Modify `TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck` (spec_alignment_test.go:2351-2368) to also require the new scoping term. Change the needle list from:

```go
for _, needle := range []string{
    "post_route_capability_check:",
    "discovery_subtype_support",
    "before_weapon_invocation",
} {
```

to:

```go
for _, needle := range []string{
    "post_route_capability_check:",
    "discovery_subtype_support",
    "before_weapon_invocation",
    "applies_to_subtypes: [creative]",
    "this check does not run",
} {
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck ./tests/spec/... -v`
Expected: FAIL — `applies_to_subtypes: [creative]` and `this check does not run` are not in the current file.

**Step 3: Edit the contract**

In `internal/embed/defaults/contracts/machine/scout-routing.yaml`, replace lines 80-97:

```yaml
post_route_capability_check:
  description: >
    Runs immediately after route_decision is emitted with
    evidence_state: requires_discovery and discovery_subtype: creative, before
    the resolved discovery weapon is invoked. This is Scout's routing
    responsibility, not classic preflight — preflight runs before
    intake/routing, before discovery_subtype exists. For discovery_subtype
    values other than creative (evaluation, diagnostic, closure_evidence),
    this check does not run: resolution goes directly to
    internal_skills/ranger (native_role) and the external weapon's manifest
    is never consulted — see
    contracts/narrative/00-routing.md § Discovery Weapon Resolution by Subtype.
  timing: before_weapon_invocation
  applies_to_subtypes: [creative]
  check: >
    Read the resolved discovery weapon's skill.yaml discovery_subtype_support field.
    If it does not declare support (native or adapter) for creative,
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

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck ./tests/spec/... -v`
Expected: PASS

Also re-run the narrative-side capability check test to make sure it's unaffected:
Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestRoutingContractDefinesPostRouteCapabilityCheck ./tests/spec/... -v`
Expected: PASS (00-routing.md's existing "Post-Route Capability Check" section text is untouched by this task; Task 2 only added a new section after it)

**Step 5: Commit**

```bash
git add internal/embed/defaults/contracts/machine/scout-routing.yaml tests/spec/spec_alignment_test.go
git commit -m "fix(scout-routing): scope post_route_capability_check to creative subtype only"
```

---

### Task 5: Remove false adapter claims from brainstorming's manifest

**Files:**
- Modify: `internal/embed/defaults/skills/brainstorming/skill.yaml:18-22`
- Test: `tests/spec/spec_alignment_test.go` (rewrite `TestBrainstormingProviderDeclaresSubtypeSupport`, currently lines 2163-2193)

**Step 1: Write the failing test**

Replace `TestBrainstormingProviderDeclaresSubtypeSupport` (spec_alignment_test.go:2163-2193) with:

```go
// TestBrainstormingProviderDeclaresOnlyCreativeSubtypeSupport verifies the
// brainstorming provider manifest declares creative-subtype support only.
// It must NOT claim evaluation/diagnostic/closure_evidence adapter support:
// those subtypes bypass this weapon entirely and resolve to
// internal_skills/ranger (see 00-routing.md § Discovery Weapon Resolution by
// Subtype). A prior version of this test required the adapter claims —
// that assumption was falsified by a live invocation showing brainstorming's
// own SKILL.md has no adaptive behavior for those subtypes at all.
func TestBrainstormingProviderDeclaresOnlyCreativeSubtypeSupport(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skills", "brainstorming", "skill.yaml")
	content := readFile(t, path)
	for _, needle := range []string{
		"canonical_role: ranger",
		"provider_class: rankeado",
		"risk_score: write_analysis",
		"discovery_subtype_support:",
		"creative: native",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing brainstorming weapon term %q", path, needle)
		}
	}
	for _, forbidden := range []string{
		"evaluation: adapter",
		"diagnostic: adapter",
		"closure_evidence: adapter",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s still declares false subtype support %q — these subtypes now bypass this weapon entirely", path, forbidden)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestBrainstormingProviderDeclaresOnlyCreativeSubtypeSupport ./tests/spec/... -v`
Expected: FAIL — current manifest still has the three `adapter` lines.

**Step 3: Edit the manifest**

In `internal/embed/defaults/skills/brainstorming/skill.yaml`, replace lines 18-22:

```yaml
discovery_subtype_support:
  creative: native
```

(Removes the `evaluation: adapter`, `diagnostic: adapter`, `closure_evidence: adapter` lines entirely.)

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestBrainstormingProviderDeclaresOnlyCreativeSubtypeSupport ./tests/spec/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/embed/defaults/skills/brainstorming/skill.yaml tests/spec/spec_alignment_test.go
git commit -m "fix(brainstorming): remove false evaluation/diagnostic/closure_evidence adapter claims"
```

---

### Task 6: Retire and replace the `brainstorming_diagnostic_not_blocked_by_standalone_creative_first` fixture scenario

**Files:**
- Modify: `internal/embed/defaults/contracts/tests/preflight.test.yaml:67-76`
- Modify: `tests/spec/spec_alignment_test.go:896-912` (`TestPreflightProviderManifestIsSlotAuthority`)

**Context:** This fixture scenario currently exercises `discovery_subtype: diagnostic` against `brainstorming` and asserts a handoff `target=ranger weapon=brainstorming subtype=diagnostic`. After Task 5, `diagnostic` never reaches `brainstorming` at all — it resolves straight to `internal_skills/ranger`. The underlying principle the scenario protects (`provider_manifest_is_slot_authority`: a weapon's standalone `SKILL.md` "looking" creative-first must not cause `role_invocation_failed` at preflight) is still valid, but needs to be exercised with a subtype `brainstorming` is actually still invoked for (`creative`), plus a new scenario documenting the bypass itself.

**Step 1: Write the failing test**

Modify `TestPreflightProviderManifestIsSlotAuthority` in `tests/spec/spec_alignment_test.go` (lines 896-912). Change the `testFiles` needle loop from:

```go
for _, path := range testFiles {
    content := readFile(t, path)
    for _, needle := range []string{
        "provider_manifest_is_slot_authority",
        "brainstorming_diagnostic_not_blocked_by_standalone_creative_first",
        "subtype=diagnostic",
    } {
        if !strings.Contains(content, needle) {
            t.Fatalf("%s missing provider authority preflight test term %q", path, needle)
        }
    }
}
```

to:

```go
for _, path := range testFiles {
    content := readFile(t, path)
    for _, needle := range []string{
        "provider_manifest_is_slot_authority",
        "brainstorming_creative_not_blocked_by_standalone_creative_first",
        "subtype=creative",
        "diagnostic_subtype_bypasses_external_weapon_for_native_ranger",
        "weapon=internal_skills/ranger",
    } {
        if !strings.Contains(content, needle) {
            t.Fatalf("%s missing provider authority preflight test term %q", path, needle)
        }
    }
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestPreflightProviderManifestIsSlotAuthority ./tests/spec/... -v`
Expected: FAIL — the old scenario name/subtype are still in the fixture; the new terms aren't yet.

**Step 3: Edit the fixture**

In `internal/embed/defaults/contracts/tests/preflight.test.yaml`, replace the `brainstorming_diagnostic_not_blocked_by_standalone_creative_first` scenario (lines 67-76) with two scenarios:

```yaml
  - name: brainstorming_creative_not_blocked_by_standalone_creative_first
    given:
      discovery_provider: brainstorming
      provider_manifest_path: skill_root/skills/brainstorming/skill.yaml
      provider_manifest_risk_score: write_analysis
      discovery_subtype: creative
      standalone_skill_md_style: creative-first
    expect:
      no_emit: "[Strategist] phase=preflight status=blocked reason=role_invocation_failed slot=discovery provider=brainstorming"
      handoff: "[Strategist] phase=intake status=handoff component=scout target=ranger weapon=brainstorming subtype=creative"

  - name: diagnostic_subtype_bypasses_external_weapon_for_native_ranger
    given:
      discovery_provider: brainstorming
      discovery_subtype: diagnostic
    expect:
      note: "external weapon manifest is never consulted for this subtype — see 00-routing.md § Discovery Weapon Resolution by Subtype"
      handoff: "[Strategist] phase=intake status=handoff component=scout target=ranger weapon=internal_skills/ranger subtype=diagnostic"
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build-cache go test -tags=spec -run TestPreflightProviderManifestIsSlotAuthority ./tests/spec/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/embed/defaults/contracts/tests/preflight.test.yaml tests/spec/spec_alignment_test.go
git commit -m "test(preflight): replace diagnostic-subtype brainstorming fixture with creative-subtype + native-bypass scenarios"
```

---

### Task 7: Full verification sweep

**Files:** none (verification only)

**Step 1: Run the full spec suite**

Run: `make spec` (equivalent to `GOCACHE=/tmp/go-build-cache go test -race -tags=spec ./tests/spec/...`)
Expected: PASS, all tests including the 6 new/modified ones above and every pre-existing test in `spec_alignment_test.go` and `corpus_lint_test.go` (in particular re-check `TestEntryDocRuntimeReferencesResolve` and `TestCorpusHasNoDotlessSourceTreePaths` — both walk every `.md`/`.yaml` file under `internal/embed/defaults/`, so the new prose added in Tasks 1-3 must not introduce a dot-less `strategist/...` path reference or an unresolvable `.strategist/...` reference).

**Step 2: Run the full non-spec test suite**

Run: `make test` (equivalent to `GOCACHE=/tmp/go-build-cache go test -race $(go list ./... | grep -v '/testutil')`)
Expected: PASS — this plan does not touch any `.go` production code (only `internal/embed/defaults/*` markdown/YAML and `tests/spec/spec_alignment_test.go`), so no regression is expected here, but this confirms nothing else was accidentally broken.

**Step 3: Rebuild the binary**

Run: `make build`
Expected: succeeds — embedding the changed files requires a rebuild since `go:embed all:defaults` bakes the corpus into the binary.

**Step 4: Verify propagation into this workspace's `.strategist/`**

This workspace's `.strategist/` was installed from a previous build and will NOT reflect these changes until recompiled — per the design doc's Canonical Scope constraint, do not hand-edit `.strategist/agent-protocol.md`; regenerate it instead:

Run: `./bin/strategist compile --root .strategist` (equivalent to `make compile-skill`, pointed at this workspace's `.strategist/`)

Then confirm the regenerated file reflects the new conditional rule:

Run: `grep -n "Discovery Routing\|internal_skills/ranger" .strategist/agent-protocol.md`
Expected: both terms present in the regenerated `.strategist/agent-protocol.md` §3.

**Step 5: No commit** (verification-only task)

---

## Notes for whoever executes this plan

- Every `git commit` step above is written per the plan-template convention but must NOT be run without a separate explicit go-ahead — the user who approved this plan asked that no git mutation commands be applied automatically in the session that produced it.
- Tasks 1-6 are independent of each other in principle (different files), but Task 5 (manifest) and Task 6 (fixture) are causally linked — the fixture only needs replacing because of the manifest change, so keep those two in sequence if executing non-linearly.
- Task 7 must run last, after all six content tasks, and is the actual acceptance gate from the design doc (`.analysis/pending/2026-07-26-discovery-subtype-native-routing-design.md` § Acceptance / manual test).
