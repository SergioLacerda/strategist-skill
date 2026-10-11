# Generic Execution Resolver Registry Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the provider-specific flat execution-target bridge with a generic resolver registry and immutable hierarchical snapshot that supports multiple independent ADR/Runbook groups while keeping Sniper provider-neutral.

**Architecture:** Introduce domain-owned ExecutionSnapshot, ExecutionGroup, ExecutionArtifact, and ResolverKey types. A mission execution registry resolves accepted intents by kind + origin; resolver adapters perform provider-specific preparation and return normalized groups. The execution entry persists the snapshot, while Sniper and completion/claims code consume only its generic artifacts and group completion state.

**Tech Stack:** Go, JSON durable execution entries, append-only JSONL telemetry, Cobra mission commands, Testify, embedded canonical YAML/Markdown contracts.

---

## Constraints and invariants

- Work only in canonical source and Go packages. Do not regenerate .strategist in this change; a parallel task owns runtime regeneration.
- Preserve unrelated staged, unstaged, renamed, and untracked work in the dirty checkout. Do not reset, stage, commit, or broadly clean files.
- The snapshot must not contain a Weapon name or provider-specific selector.
- Resolver lookup is by kind + origin; missing and duplicate registrations fail closed.
- Snapshot assembly is atomic. Once persisted, groups materialize independently.
- all_required is the only group completion policy in this wave: every primary, sidecar, and supporting artifact in a group is required.
- Legacy flat ExecutionTarget entries normalize to one-artifact groups without reading plan prose or expanding targets.
- The Sniper must never discover or expand artifacts from tasks.md, filenames, OpenSpec, Runbook, ADR, or Weapon names. It may use a generic artifact completion-source marker for the refined-package checkbox proof.

### Task 1: Define the hierarchical execution domain model

**Files:**
- Create: internal/domain/execution_snapshot.go
- Create: internal/domain/execution_snapshot_test.go
- Modify: internal/domain/execution_target.go
- Modify: internal/domain/execution_target_test.go

**Step 1: Write failing domain tests**

Add tests for:

- JSON round-trip of a snapshot containing two groups;
- a group containing a primary artifact and a sidecar;
- validation of stable non-empty group and artifact IDs;
- validation of project-relative artifact paths;
- rejection of duplicate group IDs and duplicate artifact IDs within a group;
- rejection of an empty group;
- rejection of unsupported completion policy values;
- rejection of a side-quest group without side_quest_id;
- acceptance of future kind values without hardcoded kind rejection;
- normalization of a legacy ExecutionTarget into one artifact in one group.

Use explicit test fixtures with two Runbook groups and one ADR group to prove that the model supports lists rather than a single special target.

**Step 2: Run the focused tests and verify they fail**

Run:

~~~bash
GOCACHE=/tmp/strategist-execution-domain-cache go test ./internal/domain -run 'TestExecution(Snapshot|Group|Artifact|Target)' -count=1
~~~

Expected: FAIL because the hierarchical types and validation are not present.

**Step 3: Implement the minimal domain types**

Define JSON-stable types with these semantics:

~~~go
type ResolverKey struct {
    Kind   string
    Origin string
}

type ExecutionArtifact struct {
    ID       string
    Role     string
    Path     string
    Kind     string
    Metadata map[string]string
}

type ExecutionGroup struct {
    ID               string
    Resolver         ResolverKey
    SideQuestID      string
    Kind             string
    Origin           string
    CompletionPolicy string
    Artifacts        []ExecutionArtifact
}

type ExecutionSnapshot struct {
    Version   int
    MissionID string
    Groups    []ExecutionGroup
}
~~~

Keep ExecutionTarget as a legacy compatibility type during migration. Add a conversion helper that creates one all_required group with one artifact and does not inspect any source file.

**Step 4: Run the focused tests and verify they pass**

Run the command from Step 2.

Expected: PASS.

### Task 2: Add the generic resolver registry and lifecycle contract

**Files:**
- Create: internal/mission/execution_resolver.go
- Create: internal/mission/execution_resolver_test.go
- Modify: internal/mission/execution_targets.go
- Modify: internal/mission/execution_targets_test.go

**Step 1: Write failing registry tests**

Cover:

- registration and lookup by ResolverKey{Kind, Origin};
- duplicate registration returning execution_resolver_duplicate;
- blank key rejection;
- missing lookup returning execution_resolver_not_registered with kind and origin;
- resolver output validation before it can be persisted;
- deterministic retry returning the same group and artifact identities;
- one resolver returning multiple groups.

Use a test resolver with no Weapon field or provider-specific input.

**Step 2: Run the registry tests and verify they fail**

Run:

~~~bash
GOCACHE=/tmp/strategist-resolver-cache go test ./internal/mission -run 'Test(ExecutionResolver|ResolverRegistry|ExecutionResolution)' -count=1
~~~

Expected: FAIL because the registry and resolver interfaces do not exist.

**Step 3: Implement the registry and resolver contract**

Add the domain-neutral contract:

~~~go
type ExecutionResolveInput struct {
    MissionID   string
    SideQuestID string
    Kind        string
    Origin      string
    Payload     map[string]any
}

type ExecutionResolver interface {
    Key() domain.ResolverKey
    Resolve(context.Context, ExecutionResolveInput) (domain.ExecutionResolution, error)
}
~~~

The registry must copy or freeze its resolver map after construction, validate all output groups, and expose stable typed errors. Do not add a Weapon field to the input, registry key, group, or snapshot.

**Step 4: Replace the fixed side-quest loop with registry dispatch**

Refactor ResolveExecutionTargets into a compatibility-facing resolver that:

- turns explicit refined-package paths into a documentation/refined_package intent;
- enumerates accepted side-quest records through the existing side-quest boundary rather than embedding OA-ADR/OA-RUNBOOK branches in the generic loop;
- dispatches every intent through the registry;
- assembles all returned groups before returning;
- rejects the whole assembly if any resolver is missing or invalid;
- preserves stable ordering and deduplicates only by stable group/artifact identity, never by a guessed path.

Retain a compatibility projection named ExecutionTargetPaths for existing telemetry callers, implemented as a projection of the new snapshot rather than as a second discovery path.

**Step 5: Run the registry and existing mission tests**

Run:

~~~bash
GOCACHE=/tmp/strategist-resolver-cache go test ./internal/mission -run 'Test(ExecutionResolver|ResolveExecutionTargets|ResolverRegistry)' -count=1
~~~

Expected: the new registry tests pass. Existing OA-Runbook rejection tests must be replaced by resolver-registration behavior; no test may continue to assert an OA-specific unsupported branch.

### Task 3: Implement built-in resolver adapters

**Files:**
- Create: internal/mission/execution_resolvers_builtin.go
- Create: internal/mission/execution_resolvers_builtin_test.go
- Modify: internal/mission/adr_target.go
- Modify: internal/mission/documentation_destination.go
- Modify: internal/mission/side_quest_record.go
- Modify: internal/mission/side_quest_decide.go
- Modify: internal/mission/side_quest_test.go

**Step 1: Write failing adapter tests**

Test three registered adapters:

- refined documentation: each declared path becomes an artifact with generic completion metadata indicating the refined-package task check;
- accepted ADR: one group with the reserved ADR path and no task-checkbox requirement;
- accepted Runbook: one group containing a primary artifact and sidecar, with both paths reserved/prepared by the resolver protocol.

Also test that one accepted input can return multiple Runbook groups and that two accepted ADR inputs produce independent groups. Use stable IDs and assert that repeated resolution preserves IDs and paths.

**Step 2: Run adapter tests and verify they fail**

Run:

~~~bash
GOCACHE=/tmp/strategist-resolver-adapter-cache go test ./internal/mission -run 'Test(Builtin|Documentation|ADR|Runbook)ExecutionResolver' -count=1
~~~

Expected: FAIL because the adapters and group output do not exist.

**Step 3: Implement the adapters**

Register built-ins through one constructor/factory owned by the execution package. Keep destination and reservation details inside the corresponding resolver adapter. The generic dispatch code must not switch on OA-ADR, OA-RUNBOOK, or OA-CHEST identifiers.

For Runbook, return the main artifact and sidecar in the same group. For a list of Runbooks, return one group per Runbook. For ADRs, use the existing ADR number reservation operation but expose only the normalized group result.

If current side-quest storage can represent only one deterministic Runbook ID, add the smallest generic accepted-intent enumeration needed to preserve a list of accepted records; do not encode a Sniper or Weapon-specific workaround.

**Step 4: Run adapter tests and verify they pass**

Run the command from Step 2.

Expected: PASS.

### Task 4: Persist the immutable snapshot at execution entry

**Files:**
- Modify: cmd/strategist/mission/execution_entry.go
- Modify: cmd/strategist/mission/execution_entry_persistence.go
- Modify: cmd/strategist/mission/submit.go
- Modify: cmd/strategist/mission/execution_entry_test.go
- Modify: cmd/strategist/mission/lifecycle_test.go

**Step 1: Write failing persistence tests**

Add tests that:

- persist the full snapshot, including multiple groups and sidecars;
- reload an identical snapshot from the durable execution entry;
- reject a changed snapshot while an entry is incomplete;
- rotate only when the completed entry receives a new package/outcome;
- normalize a legacy entry containing TargetMetadata or only Targets into one-artifact groups;
- never fall back to re-reading tasks.md when a snapshot is present;
- reject partial snapshot assembly when one resolver fails.

**Step 2: Run persistence tests and verify they fail**

Run:

~~~bash
GOCACHE=/tmp/strategist-entry-cache go test ./cmd/strategist/mission -run 'Test(PrepareExecutionEntry|LoadExecution|RecoverExecution)' -count=1
~~~

Expected: FAIL because the entry stores only flat targets.

**Step 3: Persist the snapshot and retain a projection for legacy consumers**

Add a versioned snapshot field to executionEntry. Keep Targets and TargetMetadata readable for old entries, but make the new snapshot the authoritative field for new entries. Update submit preparation to pass the resolver-produced snapshot through the durable entry before state persistence.

Update reconciliation to compare the snapshot as part of the immutable entry identity. Add a loader that returns the normalized snapshot and only invokes legacy conversion for entries that do not contain the new field.

**Step 4: Run persistence tests and verify they pass**

Run the command from Step 2.

Expected: PASS.

### Task 5: Make claims and completion group-aware

**Files:**
- Modify: internal/mission/sniper_claims.go
- Modify: internal/mission/sniper_claims_test.go
- Modify: cmd/strategist/mission_completion_sniper.go
- Modify: cmd/strategist/mission_execution_invocation_test.go
- Modify: cmd/strategist/command_boundary_regression_test.go
- Modify: internal/telemetry/sniper_materialization.go
- Modify: internal/telemetry/sniper_conflict.go
- Modify: internal/telemetry/sniper_conflict_test.go

**Step 1: Write failing group-completion tests**

Cover:

- a primary artifact without its required sidecar leaves the group incomplete;
- completion of all artifacts in group A succeeds even when group B is pending;
- materialization records match snapshot artifact identity/path;
- claims are projected from the snapshot and do not reparse tasks.md;
- accepted side-quest artifacts are not double-claimed by generic claim projection;
- a snapshot with no declared artifact fails closed;
- old path-only materialization records remain readable during migration.

**Step 2: Run the focused tests and verify they fail**

Run:

~~~bash
GOCACHE=/tmp/strategist-completion-cache go test ./internal/mission ./internal/telemetry ./cmd/strategist -run 'Test(Sniper|Materialization|Claim|Group|Execution)' -count=1
~~~

Expected: FAIL because completion currently treats a flat path list as the whole contract.

**Step 3: Add generic artifact/group matching**

Use snapshot artifact IDs and paths as the matching contract. If telemetry needs additional identity, add optional execution_entry_id, group_id, and artifact_id fields to SniperMaterializationRecord; keep path-only records valid for the migration window.

Refactor claim projection to accept the normalized snapshot. Move the refined task checkbox rule behind generic artifact metadata such as completion_source=refined_tasks; do not branch on ADR, Runbook, OpenSpec, or Weapon.

Refactor verifySniperLifecycleAndTasks, materialization verification, and mission completion so they:

- load the snapshot once;
- verify every declared artifact required by the relevant group;
- require task checkbox proof only for artifacts carrying the generic marker;
- evaluate group completion with all_required;
- return stable missing-artifact/group-incomplete errors.

**Step 4: Run the focused tests and verify they pass**

Run the command from Step 2.

Expected: PASS for the new group cases and existing path-only telemetry cases.

### Task 6: Update the Sniper handoff contract without provider branches

**Files:**
- Modify: internal/embed/defaults/schemas/handoff-archivist-to-sniper.schema.yaml
- Modify: internal/embed/defaults/internal_skills/sniper/SKILL.md
- Modify: internal/embed/defaults/internal_skills/sniper/skill.yaml
- Modify: internal/embed/defaults/roles/sniper.yaml
- Modify: internal/embed/defaults/contracts/narrative/06-execution.md
- Modify: internal/embed/defaults/contracts/narrative/07-adr.md
- Modify: internal/embed/defaults/contracts/tests/sniper.test.yaml
- Modify: internal/embed/defaults/templates/agent-protocol.md
- Modify: internal/embed/defaults/skill.yaml

**Step 1: Write/update contract tests first**

Add contract assertions that:

- the Archivist-to-Sniper handoff contains the hierarchical execution snapshot;
- group and artifact IDs, roles, paths, resolver key, and completion policy are typed;
- Sniper is forbidden from inferring targets from tasks.md or plan prose;
- Sniper loops over declared artifacts and can materialize a primary/sidecar group;
- no schema or instruction mentions a Weapon-specific dispatch rule.

**Step 2: Run embedded contract tests and verify the new assertions fail**

Run:

~~~bash
GOCACHE=/tmp/strategist-embedded-contract-cache go test ./internal/embed -run 'Test(Embedded|Extractor|Sniper|Handoff)' -count=1
~~~

Expected: FAIL until the canonical contracts describe the group snapshot.

**Step 3: Update canonical Sniper instructions**

Replace the flat target input with the snapshot contract while documenting the legacy compatibility read. State that Sniper writes exactly the declared artifact paths, reports generic artifact completion, and never chooses a resolver or interprets resolver_key semantically.

Keep tasks.md limited to checkbox evidence for artifacts marked with the generic refined-task completion source. Do not add Runbook, ADR, OpenSpec, or future-Weapon branches.

**Step 4: Run embedded contract tests and lint the changed YAML/Markdown**

Run the command from Step 2, followed by:

~~~bash
git diff --check
~~~

Expected: focused embedded tests PASS and git diff --check produces no output.

Do not run make embed-skills or make embed-skills-check; runtime generation belongs to the parallel task and the worktree has an unrelated known compile blocker.

### Task 7: Add end-to-end resolver/snapshot scenarios

**Files:**
- Modify: internal/mission/execution_boundary_test.go
- Modify: internal/mission/execution_boundary_routes.go
- Modify: cmd/strategist/mission/execution_guard_test.go
- Modify: cmd/strategist/mission/lifecycle_test.go
- Modify: internal/embed/defaults/contracts/tests/sniper.test.yaml

**Step 1: Add the failing scenarios**

Build scenario fixtures for:

- one refined documentation target plus one accepted ADR;
- two Runbook groups, each with primary and sidecar;
- one group partially materialized and a second group fully materialized;
- an unregistered future kind + origin failing closed;
- a legacy flat execution entry loading as one-artifact groups;
- changed tasks.md after entry creation not changing the snapshot.

**Step 2: Run the scenarios and verify the missing behavior**

Run:

~~~bash
GOCACHE=/tmp/strategist-e2e-cache go test ./internal/mission ./cmd/strategist/mission ./internal/embed -run 'Test(ExecutionBoundary|Lifecycle|Resolver|Sniper)' -count=1
~~~

Expected: FAIL only for the new hierarchical behavior before implementation is complete.

**Step 3: Wire the smallest integration changes**

Connect the registry-created snapshot to the existing execution boundary and invocation authorization. Ensure all downstream consumers receive the same loaded snapshot instance/projection and that no consumer reopens tasks.md to discover targets.

**Step 4: Run the scenarios and verify they pass**

Run the command from Step 2.

Expected: PASS for the new scenarios and existing focused boundary tests.

### Task 8: Validate the change and hand off runtime regeneration

**Files:**
- Modify only files already listed above if validation exposes a defect.
- Do not modify .strategist generated runtime files.

**Step 1: Run formatting and focused package tests**

Run:

~~~bash
gofmt -w internal/domain/execution_snapshot.go internal/mission/execution_resolver.go internal/mission/execution_resolvers_builtin.go
GOCACHE=/tmp/strategist-execution-final-cache go test ./internal/domain ./internal/mission ./internal/telemetry ./cmd/strategist/mission ./internal/embed -count=1
~~~

Expected: PASS for the affected packages. If unrelated pre-existing failures occur, record their exact test names and output rather than declaring the repository green.

**Step 2: Run targeted race tests**

Run:

~~~bash
GOCACHE=/tmp/strategist-execution-race-cache go test -race ./internal/domain ./internal/mission ./internal/telemetry ./cmd/strategist/mission -run 'Test(Execution|Resolver|Sniper|Materialization|Claim)' -count=1
~~~

Expected: PASS, or a documented pre-existing/environment blocker.

**Step 3: Verify source hygiene**

Run:

~~~bash
git diff --check
git status --short
~~~

Expected: no whitespace errors; only intended canonical source, Go, test, and plan changes are present. Do not stage or commit.

**Step 4: Hand off runtime synchronization**

Report that canonical sources are ready for the parallel runtime regeneration task. That task may regenerate .strategist and run its own convergence checks; this implementation must not do so.

## Known validation caveats

- The dirty checkout currently has an unrelated internal/missionview to internal/usagesummary transition and a deleted/relocated usage file that can block broad package compilation. Preserve and report it.
- Existing broad mission tests have known failures around ADR numbering and legacy shadow delegation telemetry. These must remain separated from the resolver-registry validation.
- Runtime synchronization is intentionally excluded from this plan's source implementation path and is owned by the parallel task.
