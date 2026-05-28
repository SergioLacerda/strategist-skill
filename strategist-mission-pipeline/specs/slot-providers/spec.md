## ADDED Requirements

### Requirement: Slot bindings declared in roles/<config>.yaml with Scout/Engineer/Hunter as public labels
The Strategist SHALL read slot provider bindings from `roles/<config>.yaml`. Public labels are `scout`, `engineer`, and `hunter`. Internally, these map to generic contract names `discovery`, `refinement`, and `execution` for compatibility. The default config after init is `roles/default.yaml`. When SDD-integrated, the Hunter (execution) provider is overridden by `sdd_injection.execution_provider`.

```yaml
# Example roles/default.yaml (standalone):
scout: sdd-diagnose
engineer: sdd-engineer
hunter: caveman

# Example roles/sdd-mission.yaml (SDD integration):
scout: sdd-diagnose
engineer: sdd-engineer
hunter: _injected_by_sdd   # resolved from sdd_injection at runtime
```

#### Scenario: Default roles config loaded
- **WHEN** Strategist is invoked without explicit `--roles` parameter
- **THEN** `roles/default.yaml` is loaded as the slot binding config

#### Scenario: Custom roles config loaded
- **WHEN** Strategist is invoked with `--roles sdd-mission`
- **THEN** `roles/sdd-mission.yaml` is loaded as the slot binding config

#### Scenario: Hunter injected by SDD
- **WHEN** `sdd_injection.execution_provider` is `sdd-ask` (SDD integration mode)
- **THEN** Hunter slot resolves to `sdd-ask`, overriding any value in roles.yaml

#### Scenario: Hunter configured at init (standalone)
- **WHEN** no `sdd_injection` is present
- **THEN** Hunter resolves from `roles/default.yaml` hunter field

---

### Requirement: Scout and Engineer slots require read_only providers
Any provider bound to the Scout (discovery) or Engineer (refinement) slot MUST have `risk_score: read_only` and `status: active` in their `skill.yaml`.

#### Scenario: Valid read_only provider bound
- **WHEN** Scout slot is bound to a skill with `risk_score: read_only`
- **THEN** preflight passes for this slot

#### Scenario: Invalid provider bound to Scout
- **WHEN** Scout slot is bound to a skill with `risk_score: high`
- **THEN** preflight fails with `reason=slot_risk_mismatch slot=scout` and identifies the provider

---

### Requirement: Hunter slot requires controlled_write provider
The provider bound to the Hunter (execution) slot MUST have `risk_score: controlled_write` and `status: active`.

#### Scenario: Valid Hunter provider
- **WHEN** Hunter slot resolves to a skill with `risk_score: controlled_write`
- **THEN** preflight passes for Hunter slot

#### Scenario: Hunter resolved to read_only provider
- **WHEN** configured Hunter provider has `risk_score: read_only`
- **THEN** preflight fails with `reason=slot_risk_mismatch slot=hunter`

---

### Requirement: Provider resolution follows declared order
The Strategist SHALL resolve each slot provider in this order: (1) `<skill_root>/<provider>/skill.yaml`, (2) `.claude/skills/<provider>/skill.yaml`, (3) skill registry entry `skill_yaml` path (if registry present). If no path resolves, preflight fails.

#### Scenario: Provider found at primary path
- **WHEN** `<skill_root>/sdd-diagnose/skill.yaml` exists
- **THEN** Scout slot resolves without checking further paths

#### Scenario: Provider not found at any path
- **WHEN** no path yields a valid skill.yaml for the declared provider
- **THEN** preflight emits `reason=slot_provider_not_found slot=<scout|engineer|hunter>` with provider id and paths checked
