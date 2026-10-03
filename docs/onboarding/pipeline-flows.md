# Strategist Pipeline Flows

This guide explains how a workspace becomes configured, how a request moves through Strategist, and how a Weapon becomes safely bound to a Role or slot.

## 1. Install flow

Typical entry points are:

```bash
strategist install
strategist install --wizard
strategist install --silent
```

Useful controls include `--target`, `--force`, `--no-shim`, and `--strict-compile`.

The install transaction is:

```text
prepare runtime
  → write workspace configuration
  → refresh Ranked bindings
  → prepare private Ranked runtimes
  → ensure .gitignore coverage for .strategist/.compiled/
  → optionally install the user shim
  → finalize install manifest
```

The installer extracts embedded defaults into the target `.strategist/` directory. A normal compile warning does not necessarily abort installation; `--strict-compile` makes compile failure fatal and enables rollback semantics. `--no-shim` avoids writing a user-level command shim.

If a transaction step fails, the installer rolls back its own transaction. Install success means that the workspace artifacts were prepared; it does not prove that every provider can later perform authenticated live work.

## 2. Wizard flow

The wizard configures the active workspace rather than editing provider code.

```text
load catalog and compatibility graph
  → choose UI/docs/chat/code language axes
  → choose mode and .analysis base path
  → choose compatible discovery/refinement providers
  → choose treasure-chest scope
  → compute onboarding plan and Role affinity
  → resolve and activate pinned bindings
  → materialize Custom providers when selected
  → persist active.yaml and plugins.lock
```

Important rules:

- `LEVELING` is automatic for a new install; it is not a user-selected provider.
- Discovery and refinement do not silently fall back to a native or latest provider.
- Role affinity filters the provider choices shown for each slot.
- A missing compatible Weapon is a configuration failure, not an invitation to guess.
- `.strategist/plugins.lock` is written only after the selected plan is resolved.

## 3. Weapon acquisition, ROSTER, and binding

Weapon acquisition starts with a provider source and ends with a pinned runtime identity:

```text
external-skills-source/<id>/
  → prepare-embedded / make generate-embedded
  → compiled catalog and registry
  → compatibility and certification checks
  → ROSTER selection
  → plugins.lock binding
  → private runtime preparation
  → Role/slot invocation
```

`make embed-skills-check` detects generated-source drift. The supported model does not introduce an `input/weapons/` directory as a parallel authority.

Binding is fail-closed. Strategist reconciles all of the following:

| Input | Required meaning |
| --- | --- |
| Active slot intent | Which Role/slot needs a provider |
| Compiled registry | Which provider identities are known |
| `.strategist/plugins.lock` | Which version and digest are selected |
| Compatibility graph | Whether Role, slot, handoff, and runtime are compatible |
| Runtime identity | Whether the provider runtime kind and digest match |

Ranked bindings require matching provider id, version, source/binding digests, and runtime identity. Custom bindings require complete provenance and compatibility evidence. Missing or incompatible providers produce a diagnostic or `role_invocation_failed`.

There is no catalog-order, native-Role, or latest-version fallback. A static `ready` result proves configuration/readiness only; live invocation requires separate execution evidence.

## 4. FULL mission flow

FULL is the conservative default when discovery or ambiguity remains:

```text
bootstrap and preflight
  → intake
  → Scout route decision
  → context enrichment
  → Ranger discovery
  → Ranger-to-Archivist handoff challenge, when required
  → Archivist refinement
  → Approval Gate
  → optional Sniper documentation materialization
  → learning and completion report
```

### Bootstrap and intake

Preflight validates the active Strategist identity, bindings, and governance prerequisites. Intake records the request and lets Scout classify it as a general mission, implementation request, evaluation, or another governed category.

### Discovery

Ranger gathers source evidence, identifies current versus proposed behavior, records uncertainties, and produces a pending analysis artifact. The handoff challenge checks recall, boundaries, classifications, and verdict before refinement when the policy requires it.

### Refinement

Archivist consolidates the evidence into a proposal, design, tasks, and analysis package. Documentation tasks are distinct from `implementation_handoff` tasks. OpenSpec scratch output is normalized into `.analysis/refined/<mission-id>/`.

### Approval Gate

The user reviews the refined package. Acceptance authorizes only the declared documentation targets. It does not authorize source-code, test, configuration, CI, generated-runtime, Git, or landing-page changes.

### Execution and learning

Sniper claims the accepted package, scans for scope violations, materializes one documentation target at a time, writes a report, and records completion. Learning is non-blocking and does not replace the gate or execution evidence.

## 5. SHORT mission flow

SHORT is allowed only when all of these conditions hold:

- implementation or materialization intent is explicit;
- documentation targets and boundaries are already clear;
- there is no material ambiguity requiring broad discovery;
- the scope is narrow and local;
- no code or Git mutation is needed;
- broad evidence collection is unnecessary.

The route is:

```text
bootstrap/preflight
  → intake and route confirmation
  → focused context
  → targeted refinement or execution preparation
  → Approval Gate
  → approved documentation materialization
```

SHORT still requires the Approval Gate. If any prerequisite is unclear, Strategist returns to FULL rather than treating a short route as permission to skip discovery or governance.

## 6. Failure and rollback boundaries

| Boundary | Failure behavior |
| --- | --- |
| Install transaction | Roll back the install transaction; strict compile can make compile failure fatal. |
| Wizard compatibility | Stop without inventing a provider or fallback binding. |
| Weapon binding | Fail closed with a binding or invocation diagnostic. |
| Ranger handoff | Return to discovery/refinement when required evidence or classification is missing. |
| Approval Gate | Wait for accept/review/reject; never auto-accept. |
| Sniper scope scan | Stop before writing if tasks contain forbidden implementation work or undeclared targets. |
| Documentation materialization | Stop and report immediately when an out-of-scope write emerges. |

## 7. Landing-page content map

Recommended landing sections:

| Section | Reusable message | Reference |
| --- | --- | --- |
| Taxonomy | Roles own work; Weapons provide bounded capability; Tools and Feats support the governed workflow. | [Resource catalog](resource-catalog.md) |
| Setup | Install extracts a governed workspace; Wizard selects compatible, pinned providers. | This guide, install and wizard sections |
| Journey | Scout → Ranger → Archivist → Approval Gate → Sniper. | FULL flow |
| Safety | Binding is fail-closed; readiness and live invocation are different claims. | ROSTER/binding section |
| Routes | FULL is the default; SHORT is narrow and still gated. | FULL/SHORT sections |

Landing-page implementation remains a separate mission so copy review, navigation, and web changes can be approved independently.
