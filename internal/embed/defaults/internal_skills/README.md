# Internal Skills — Shape Rule

Every internal skill has a `skill.yaml`. Five of the ten additionally have a `SKILL.md`.
This is not an inconsistency to fix — it reflects two different invocation shapes.

## The rule

- **`skill.yaml` is always required.** It is the invocation contract: `id`, `version`,
  `risk_score`, `description`, `input`, `output`, plus `behavior` (the procedural steps),
  `forbidden_behaviors`, and `failure_modes`. Every internal skill, regardless of shape,
  must define these.

- **`SKILL.md` is additionally required when invoking the skill means the orchestrating
  agent adopts a distinct persona/identity for that turn** — a pipeline slot role
  (Archivist, Ranger, Sniper) or a standalone addressable capability with its own
  entry point (Scout). The tell is textual: these `SKILL.md` files
  address the invoked agent in second person ("You are Scout...", "you never perform
  discovery") and carry what `skill.yaml`'s structured fields cannot express — scope
  contracts, ordered checklists spanning a whole phase, and completion/handoff protocol.
  `skill.yaml`'s own `behavior:` list stays the terse machine-checkable summary; `SKILL.md`
  is the full agent-facing instructions for that identity.

- **Manifest-only (`skill.yaml`, no `SKILL.md`) is valid when the skill is a bounded
  computation the *current* agent (whichever persona is already active) executes inline,
  without switching identity** — classification, retrieval, assembly, scoring. Its entire
  procedural contract fits in `skill.yaml`'s `behavior` / `triage_questions` /
  `failure_modes` fields, written in third-person imperative ("Classify task_type...",
  "Load knowledge_index_path...") because there is no separate agent identity to instruct.

## What the orchestrating agent reads before invoking each shape

- **`skill.yaml` + `SKILL.md` (role slot / standalone capability):** read `skill.yaml`
  first — `risk_score` and `input`/`output` decide *whether* and *how* to invoke. Then
  hand `SKILL.md` to the invoked turn as its complete operating instructions; do not
  re-derive its procedure from `skill.yaml` alone.
- **`skill.yaml` only (inline sub-routine):** read `skill.yaml` and execute its `behavior`
  steps directly in the current turn. No separate document to load.

## Classification

| Skill | Shape | Why |
|---|---|---|
| `archivist` | `skill.yaml` + `SKILL.md` | Pipeline slot role (`skill_type: role_filler`) |
| `ranger` | `skill.yaml` + `SKILL.md` | Pipeline slot role (`skill_type: role_filler`) |
| `sniper` | `skill.yaml` + `SKILL.md` | Pipeline slot role (`skill_type: role_filler`) |
| `scout` | `skill.yaml` + `SKILL.md` | Standalone addressable capability — internal Intake Router, no `active.yaml` slot, but its own persona ("You are Scout") |
| `prompt-intake` | `skill.yaml` only | Inline classification sub-routine run by whichever agent is current; full algorithm (incl. `triage_questions`, `failure_modes`) already self-contained in `skill.yaml` |
| `context-enrichment` | `skill.yaml` only | Inline retrieval sub-routine; `behavior` fully specifies the algorithm |
| `dossier-builder` | `skill.yaml` only | Inline assembly sub-routine consuming `context-enrichment` output |
| `learning-curator` | `skill.yaml` only | Inline sub-routine; checkpoint prompt and approval logic fully specified in `behavior` |
| `response-critic` | `skill.yaml` only | Inline scoring sub-routine against a rubric |

**On the D11 concern that `prompt-intake`'s procedural behavior might be "described only in
scattered contracts":** inspected directly — it is not. `prompt-intake/skill.yaml`'s
`behavior` block already contains the complete deterministic algorithm (task_type
classification, constraint extraction, the four-step `token_strategy` inference, and the
priority-ordered economic triage gate), plus `triage_questions` and `failure_modes`. No
missing `SKILL.md` content to write — it correctly stays manifest-only under the rule
above.
