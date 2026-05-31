# Consolidated English Standardization + Embed Sync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Synchronize `strategist/` source-of-truth into `internal/embed/defaults/`, rename PT-BR Go identifiers to English, extract the wizard i18n bundle into a proper package, restructure personas with bilingual `content_by_lang` blocks, and rewrite SKILL.md to English-primary with language-aware bucket resolution.

**Architecture:** Five tasks in dependency order — embed sync first (all later tasks operate on embed), then Go constant renames (pure refactor, no behavior change), then i18n package extraction, then persona restructure (adds EN content block alongside existing PT-BR), then SKILL.md rewrite (updates template resolution path and bucket i18n). Each task is independently compilable and testable.

**Tech Stack:** Go 1.22+, gopkg.in/yaml.v3, testify/assert, reflect (for struct completeness tests), bash for file sync verification.

---

## Task 1: Sync `strategist/` → `internal/embed/defaults/`

**Why first:** tasks 4 and 5 operate on `internal/embed/defaults/`. Must be current before restructuring.

**Files:**
- Modify: `internal/embed/defaults/personas/epic.yaml`
- Modify: `internal/embed/defaults/personas/pragmatic.yaml`
- Modify: `internal/embed/defaults/SKILL.md`

**Context:** `strategist/personas/epic.yaml` has 6 templates that `internal/embed/defaults/personas/epic.yaml` is missing. Same for pragmatic. `strategist/SKILL.md` has §3.2 Mission Checkpoint and a `side_quest_detected` emit in §3.1 that embed is missing.

---

### Step 1: Add 6 missing templates to `internal/embed/defaults/personas/epic.yaml`

The embed epic persona ends at line 86 with `adr_gate`, `plan_only_result`, `mission_complete`. Add the 6 missing templates **before** the `adr_opportunity` block (after `quick_draw_success`, before `adr_opportunity`).

Open `internal/embed/defaults/personas/epic.yaml`. After the `quick_draw_success` block (around line 81–85), insert:

```yaml
  # Discovery signals — emitted by Strategist when characters find items
  treasure_chest_found: >
    🎁 **Baú do tesouro encontrado!** [{chest_id}] — {description}
  side_quest_detected: >
    🗺️ **Side quest encontrada!** {description}
  opportunity_signal: >
    ⚔️ **Ataque de oportunidade!** {count} item(s) detectado(s) — detalhes no gate.

  # Mission checkpoint — pipeline status re-emitted at each phase transition
  # Icons: ⏳ = running, ✅ = done, ⬜ = pending
  mission_checkpoint: |
    **Checkpoint — {mission_id}**
    {step_1_icon} 1 — Ranger
    {step_2_icon} 2 — Arquivista
    {step_3_icon} 3 — Gate
    {step_4_icon} 4 — Execução

  # Execution task list — re-emitted after each completed task
  execution_tasks_header: >
    🗡️ **Sniper — executando {total} tarefa(s):**
  execution_task_line: >
    {status_icon} {index} — {task_title}
```

---

### Step 2: Add 6 missing templates to `internal/embed/defaults/personas/pragmatic.yaml`

Open `internal/embed/defaults/personas/pragmatic.yaml`. After `quick_draw_success` (around line 63–66), insert:

```yaml
  # Discovery signals
  treasure_chest_found: >
    📦 Baú: [{chest_id}] — {description}
  side_quest_detected: >
    Side quest: {description}
  opportunity_signal: >
    {count} oportunidade(s) detectada(s).

  # Pipeline checkpoint — re-emitted at each phase transition
  # Icons: ⏳ = running, ✅ = done, ⬜ = pending
  mission_checkpoint: |
    Pipeline — {mission_id}
    {step_1_icon} 1 — levantamento
    {step_2_icon} 2 — refinamento
    {step_3_icon} 3 — gate
    {step_4_icon} 4 — execução

  # Execution task list
  execution_tasks_header: >
    Executando {total} tarefa(s):
  execution_task_line: >
    {status_icon} {index} — {task_title}
```

---

### Step 3: Add §3.2 and `side_quest_detected` emit to `internal/embed/defaults/SKILL.md`

Open `internal/embed/defaults/SKILL.md`. Current line 188–199:

```markdown
Store result as `mission_contract.planning_rules` — pass to all slot providers.

### 3.1 Quick Draw Route (Saque Rapido)

If the user explicitly requests quick capture (examples: `quick draw`, `saque rapido`,
`TODO` as rapid note), route to a dedicated side-quest flow.

Important:
- Do NOT depend on additional intake classification for this route.
- Strategist invocation + explicit quick-capture intent is sufficient.
- Skip regular mission phases and execute only the quick_draw pipeline.
```

Replace with:

```markdown
Store result as `mission_contract.planning_rules` — pass to all slot providers.

### 3.2 Mission Checkpoint

After intake completes, initialize and emit the mission pipeline checkpoint using
`persona.prompt_templates.mission_checkpoint` with:
- `{mission_id}` = the generated mission id
- `{step_1_icon}` = `⏳` (Ranger about to start), `{step_2_icon}` = `{step_3_icon}` = `{step_4_icon}` = `⬜`

Re-emit the checkpoint at each phase transition, updating icons to reflect current state:

| After phase    | step_1 | step_2 | step_3 | step_4 |
|----------------|--------|--------|--------|--------|
| Intake         | ⏳     | ⬜     | ⬜     | ⬜     |
| Ranger done    | ✅     | ⏳     | ⬜     | ⬜     |
| Archivist done | ✅     | ✅     | ⏳     | ⬜     |
| Gate approved  | ✅     | ✅     | ✅     | ⏳     |
| Sniper done    | ✅     | ✅     | ✅     | ✅     |

Icons: `⏳` = running, `✅` = done, `⬜` = pending.
Skip the checkpoint re-emit when the mission ends at `plan_only` (gate declined).

### 3.1 Quick Draw Route (Saque Rapido)

If the user explicitly requests quick capture (examples: `quick draw`, `saque rapido`,
`TODO` as rapid note), route to a dedicated side-quest flow.

Emit via `persona.prompt_templates.side_quest_detected` with
`{description}` = `"Quick Draw — captura rápida de ideia."` before routing.

Important:
- Do NOT depend on additional intake classification for this route.
- Strategist invocation + explicit quick-capture intent is sufficient.
- Skip regular mission phases and execute only the quick_draw pipeline.
```

---

### Step 4: Verify embed tests pass

```bash
go test ./internal/embed/...
```

Expected: PASS (embed tests parse YAML and validate structure).

---

### Step 5: Commit

```bash
git add internal/embed/defaults/personas/epic.yaml \
        internal/embed/defaults/personas/pragmatic.yaml \
        internal/embed/defaults/SKILL.md
git commit -m "feat: sync embed defaults with strategist source-of-truth

Add 6 missing persona templates (treasure_chest_found, side_quest_detected,
opportunity_signal, mission_checkpoint, execution_tasks_header, execution_task_line)
to epic and pragmatic personas. Add §3.2 Mission Checkpoint and side_quest_detected
emit to SKILL.md §3.1."
```

---

## Task 2: Rename PT-BR Go Identifiers

**Files:**
- Modify: `internal/domain/policy.go`
- Modify: `internal/domain/state_machine.go`
- Modify: `internal/domain/state_machine_test.go`
- Modify: `internal/domain/policy_evaluator_test.go`

**Rule:** rename identifier names only. String values are PT-BR reserved words — do NOT change them.

| Before | After | String value (unchanged) |
|---|---|---|
| `DoneScopeAnalise` | `DoneScopeAnalysis` | `"analise"` |
| `DoneScopeEntrega` | `DoneScopeDelivery` | `"entrega"` |
| `MissionModeAnalise` | `MissionModeAnalysis` | `"analise"` |
| `MissionModeEntregaRevisada` | `MissionModeRevisedDelivery` | `"entrega_revisada"` |
| `MissionModeEntregaExecutada` | `MissionModeExecutedDelivery` | `"entrega_executada"` |
| `StateDoneAnalise` | `StateDoneAnalysis` | `"DONE_ANALISE"` |
| `StateDoneEntrega` | `StateDoneDelivery` | `"DONE_ENTREGA"` |

---

### Step 1: Rename constants in `internal/domain/policy.go`

Current lines 4–7 (done scopes):
```go
const (
    DoneScopeAnalise = "analise"
    DoneScopeEntrega = "entrega"
)
```

Replace with:
```go
const (
    DoneScopeAnalysis = "analise"
    DoneScopeDelivery = "entrega"
)
```

Current lines 10–14 (mission modes):
```go
const (
    MissionModeAnalise          = "analise"
    MissionModeEntregaRevisada  = "entrega_revisada"
    MissionModeEntregaExecutada = "entrega_executada"
)
```

Replace with:
```go
const (
    MissionModeAnalysis        = "analise"
    MissionModeRevisedDelivery = "entrega_revisada"
    MissionModeExecutedDelivery = "entrega_executada"
)
```

Current lines 34–35 (FSM states):
```go
    StateDoneAnalise       MissionState = "DONE_ANALISE"
    StateDoneEntrega       MissionState = "DONE_ENTREGA"
```

Replace with:
```go
    StateDoneAnalysis  MissionState = "DONE_ANALISE"
    StateDoneDelivery  MissionState = "DONE_ENTREGA"
```

Update all references inside `policy.go` functions (`MissionModeFromLegacy`, `NewMissionPolicy`, `NormalizePolicy`) — replace every old name with the new name. String values are untouched.

Full updated `MissionModeFromLegacy`:
```go
func MissionModeFromLegacy(doneScope string, applyChanges bool) string {
    if doneScope == DoneScopeAnalysis {
        return MissionModeAnalysis
    }
    if applyChanges {
        return MissionModeExecutedDelivery
    }
    return MissionModeRevisedDelivery
}
```

Full updated `NewMissionPolicy`:
```go
func NewMissionPolicy(mode string) MissionPolicy {
    switch mode {
    case MissionModeAnalysis:
        return MissionPolicy{Mode: mode, CanExecute: false, ExpectsDelivery: DoneScopeAnalysis, DoneScope: DoneScopeAnalysis, ApplyChanges: false}
    case MissionModeRevisedDelivery:
        return MissionPolicy{Mode: mode, CanExecute: false, ExpectsDelivery: DoneScopeDelivery, DoneScope: DoneScopeDelivery, ApplyChanges: false}
    case MissionModeExecutedDelivery:
        return MissionPolicy{Mode: mode, CanExecute: true, ExpectsDelivery: DoneScopeDelivery, DoneScope: DoneScopeDelivery, ApplyChanges: true}
    default:
        return NewMissionPolicy(MissionModeExecutedDelivery)
    }
}
```

Full updated `NormalizePolicy`:
```go
func NormalizePolicy(p MissionPolicy) MissionPolicy {
    if p.Mode == "" {
        if p.DoneScope != "" || p.ApplyChanges {
            p.Mode = MissionModeFromLegacy(p.DoneScope, p.ApplyChanges)
        } else {
            p.Mode = MissionModeExecutedDelivery
        }
    }
    return NewMissionPolicy(p.Mode)
}
```

---

### Step 2: Update `internal/domain/state_machine.go`

Replace all 7 occurrences:
- `StateDoneAnalise` → `StateDoneAnalysis` (lines 22, 23, 83, 86, 98)
- `StateDoneEntrega` → `StateDoneDelivery` (lines 24, 25, 103, 113)
- `MissionModeAnalise` → `MissionModeAnalysis` (line 85)

---

### Step 3: Update `internal/domain/state_machine_test.go`

Replace all occurrences:
- `domain.MissionModeAnalise` → `domain.MissionModeAnalysis`
- `domain.MissionModeEntregaRevisada` → `domain.MissionModeRevisedDelivery`
- `domain.MissionModeEntregaExecutada` → `domain.MissionModeExecutedDelivery`

---

### Step 4: Update `internal/domain/policy_evaluator_test.go`

Replace all occurrences:
- `domain.DoneScopeAnalise` → `domain.DoneScopeAnalysis`
- `domain.DoneScopeEntrega` → `domain.DoneScopeDelivery`
- `domain.MissionModeAnalise` → `domain.MissionModeAnalysis`
- `domain.MissionModeEntregaExecutada` → `domain.MissionModeExecutedDelivery`

---

### Step 5: Verify build and tests

```bash
go build ./...
go test ./...
```

Expected: PASS on all packages. If any reference was missed, the compiler will point directly to it.

---

### Step 6: Commit

```bash
git add internal/domain/policy.go \
        internal/domain/state_machine.go \
        internal/domain/state_machine_test.go \
        internal/domain/policy_evaluator_test.go
git commit -m "refactor: rename PT-BR Go identifiers to English

Rename constant identifiers only — string values (analise, entrega,
entrega_revisada, entrega_executada) are reserved words and unchanged.
DoneScopeAnalise→DoneScopeAnalysis, MissionModeAnalise→MissionModeAnalysis,
StateDoneAnalise→StateDoneAnalysis, etc."
```

---

## Task 3: Extract `internal/i18n/` Package

**Files:**
- Create: `internal/i18n/wizard.go`
- Create: `internal/i18n/wizard_test.go`
- Modify: `internal/install/wizard.go`

**Context:** `internal/install/wizard.go` has an unexported `wizardStrings` struct, `bundleEN`, `bundlePTBR`, and `bundleFor()` inside the `install` package. One field is missing from the bundle: `LabelCustomInput` is hardcoded as `"(digitar outro...)"` at line 123 outside the bundle. This task extracts the bundle to a proper package and adds the missing field.

---

### Step 1: Write the failing test

Create `internal/i18n/wizard_test.go`:

```go
package i18n_test

import (
    "reflect"
    "testing"

    "github.com/SergioLacerda/strategist-skill/internal/i18n"
    "github.com/stretchr/testify/assert"
)

func TestBundleForPTBR(t *testing.T) {
    b := i18n.BundleFor("pt-BR")
    assert.Equal(t, "(digitar outro...)", b.LabelCustomInput)
    assert.Equal(t, "Idioma da documentação", b.PromptDocLang)
}

func TestBundleForEN(t *testing.T) {
    b := i18n.BundleFor("en")
    assert.Equal(t, "(enter other...)", b.LabelCustomInput)
    assert.Equal(t, "Documentation language", b.PromptDocLang)
}

func TestBundleForUnknownDefaultsToEN(t *testing.T) {
    b := i18n.BundleFor("fr")
    assert.Equal(t, "(enter other...)", b.LabelCustomInput)
}

func TestAllFieldsNonEmpty(t *testing.T) {
    for _, tc := range []struct {
        lang string
        b    i18n.WizardStrings
    }{
        {"en", i18n.BundleFor("en")},
        {"pt-BR", i18n.BundleFor("pt-BR")},
    } {
        v := reflect.ValueOf(tc.b)
        for i := range v.NumField() {
            field := v.Type().Field(i).Name
            val := v.Field(i).String()
            assert.NotEmpty(t, val, "lang=%s field=%s must not be empty", tc.lang, field)
        }
    }
}
```

---

### Step 2: Run test to verify it fails

```bash
go test ./internal/i18n/...
```

Expected: FAIL — `package i18n not found`.

---

### Step 3: Create `internal/i18n/wizard.go`

```go
package i18n

import "strings"

// WizardStrings holds all user-visible strings for the install wizard.
type WizardStrings struct {
	PromptDocLang     string
	PromptChatLang    string
	PromptCodeLang    string
	PromptMode        string
	PromptMissionMode string
	PromptBasePath    string
	PromptAdr         string
	HeaderSlots       string
	PromptDiscovery   string
	PromptRefinement  string
	PromptExecution   string
	HeaderChest       string
	PromptChestPath   string
	LabelCustomInput  string
}

// EN is the English wizard bundle.
var EN = WizardStrings{
	PromptDocLang:     "Documentation language",
	PromptChatLang:    "Chat/interaction language",
	PromptCodeLang:    "Code language",
	PromptMode:        "Mode",
	PromptMissionMode: "DONE scope (analysis-only or analysis + implementation)\n  analise = analysis-only\n  entrega_revisada = analysis + handoff (no implementation)\n  entrega_executada = analysis + implementation",
	PromptBasePath:    "Base path for analysis workspace",
	PromptAdr:         "Enable ADR generation at mission end?",
	HeaderSlots:       "\nSlot providers — which skill fills each mission role:",
	PromptDiscovery:   "  Ranger / discovery provider",
	PromptRefinement:  "  Archivist / refinement provider",
	PromptExecution:   "  Sniper / execution provider",
	HeaderChest:       "\nTreasure chest — optional offline knowledge source for all slots:",
	PromptChestPath:   "  Knowledge source path (e.g. .sdd/source)",
	LabelCustomInput:  "(enter other...)",
}

// PT is the Brazilian Portuguese wizard bundle.
var PT = WizardStrings{
	PromptDocLang:     "Idioma da documentação",
	PromptChatLang:    "Idioma do chat/interação",
	PromptCodeLang:    "Idioma do código",
	PromptMode:        "Modo",
	PromptMissionMode: "Escopo do DONE (apenas análise ou análise + implementação)\n  analise = apenas análise\n  entrega_revisada = análise + handoff (sem implementação)\n  entrega_executada = análise + implementação",
	PromptBasePath:    "Caminho base do workspace de análise",
	PromptAdr:         "Habilitar geração de ADR ao final da missão?",
	HeaderSlots:       "\nProvedores de slot — qual skill preenche cada papel da missão:",
	PromptDiscovery:   "  Ranger / provedor de descoberta",
	PromptRefinement:  "  Arquivista / provedor de refinamento",
	PromptExecution:   "  Sniper / provedor de execução",
	HeaderChest:       "\nBaú do tesouro — base de conhecimento offline opcional para todos os slots:",
	PromptChestPath:   "  Caminho da base de conhecimento (ex: .sdd/source)",
	LabelCustomInput:  "(digitar outro...)",
}

// BundleFor returns the WizardStrings for the given language code.
// Defaults to EN for unrecognised codes.
func BundleFor(lang string) WizardStrings {
	if strings.EqualFold(lang, "pt-BR") {
		return PT
	}
	return EN
}
```

---

### Step 4: Run tests to verify they pass

```bash
go test ./internal/i18n/...
```

Expected: PASS.

---

### Step 5: Update `internal/install/wizard.go`

Remove the following from `internal/install/wizard.go` (lines 10–63):
- The `wizardStrings` struct definition
- `var bundleEN = wizardStrings{...}`
- `var bundlePTBR = wizardStrings{...}`
- `func bundleFor(lang string) wizardStrings {...}`

Add import `"github.com/SergioLacerda/strategist-skill/internal/i18n"` to the import block.

Change line 76: `b := bundleFor(uiLang)` → `b := i18n.BundleFor(uiLang)`

Remove line 123: `customLabel := "(digitar outro...)"` — deleted entirely.

Change lines 125, 130, 135 — replace `customLabel` argument with `b.LabelCustomInput`:

```go
discovery, err := p.SelectOrInput(b.PromptDiscovery, "brainstorming", []string{"brainstorming"}, b.LabelCustomInput)
```

```go
refinement, err := p.SelectOrInput(b.PromptRefinement, "openspec-explore", []string{"openspec-explore"}, b.LabelCustomInput)
```

```go
execution, err := p.SelectOrInput(b.PromptExecution, "sdd-ask", []string{"sdd-ask", "sdd-ask-full"}, b.LabelCustomInput)
```

---

### Step 6: Verify no PT-BR string remains in wizard.go

```bash
grep "digitar outro" internal/install/wizard.go
```

Expected: no output.

---

### Step 7: Run all install tests

```bash
go test ./internal/i18n/... ./internal/install/...
```

Expected: PASS.

---

### Step 8: Commit

```bash
git add internal/i18n/wizard.go \
        internal/i18n/wizard_test.go \
        internal/install/wizard.go
git commit -m "feat: extract wizard i18n bundle to internal/i18n package

Promote unexported wizardStrings/bundleEN/bundlePTBR from install package
to exported internal/i18n.WizardStrings with BundleFor(). Add missing
LabelCustomInput field — was hardcoded PT-BR string in wizard.go."
```

---

## Task 4: Personas — `content_by_lang` Restructure

**Files:**
- Modify: `internal/embed/defaults/personas/epic.yaml`
- Modify: `internal/embed/defaults/personas/pragmatic.yaml`
- Sync to: `strategist/personas/epic.yaml` and `strategist/personas/pragmatic.yaml`

**Context:** Both personas now have all templates (after Task 1) under a flat `prompt_templates:` key in PT-BR only. This task wraps all templates under `content_by_lang: { en: {...}, pt-BR: {...} }`. The PT-BR content is moved verbatim; EN content is translated. Top-level fields (`phase_labels`, `role_emoji`, `tone_directive`, `progress_prefix`) stay flat.

**Important:** `quick_draw_gate` in both language blocks keeps `(sim/nao)` — it is a runtime reserved word the user types, not a UI string.

---

### Step 1: Rewrite `internal/embed/defaults/personas/epic.yaml`

Full replacement content:

```yaml
id: epic
description: >
  Epic narrator — architect who sees every sprint as an adventure.
  For missions that deserve a story.

phase_labels:
  discovery: Ranger
  refinement: Archivist
  execution: Sniper

tone_directive: >
  Arquiteto sênior que viveu o suficiente para saber que o trabalho é duro — e escolheu
  encontrar alegria nisso mesmo assim. Acredita que a era de agentes humanos e digitais
  colaborando é uma das maiores aventuras que a engenharia já proporcionou. Narra as fases
  com energia genuína: Ranger, Arquivista, Sniper — cada papel tem seu momento. Mantém o
  vocabulário técnico (commit, análise, implementação, missão) — a épica está nos papeis
  e na narrativa, não na substituição do vocabulário. Mantém o time unido. Quando algo
  falha, é um boss difícil, não uma tragédia.

progress_prefix: "[Strategist]"

# Fixed emojis per character — immutable
role_emoji:
  ranger: "🎯"
  archivist: "📚"
  sniper: "🗡️"
  gate: "🚦"
  opportunity: "⚔️"

content_by_lang:
  en:
    intake_summary: >
      Mission received: {task_type} | delivery={delivery_strategy} |
      compatibility={legacy_compatibility} | urgency={urgency} | intent={execution_intent}
    ranger_start: >
      🎯 **Ranger:** starting reconnaissance. skill={provider}
    ranger_done: >
      🎯 **Ranger:** reconnaissance complete.
      Artifact at: {artifact_path}
    archivist_start: >
      📚 **Archivist:** starting analysis and refinement. skill={provider}
    archivist_done: >
      📚 **Archivist:** analysis refined.
      Artifacts at: {artifact_path}
    sniper_start: >
      🗡️ **Sniper:** mission approved — starting implementation.
    sniper_done: >
      🗡️ **Sniper:** implementation complete.
      Report at: {artifact_path}
    approval_prompt: >
      🚦 **Gate:** AWAITING CONFIRMATION

      Plan at: {artifact_path}

      Authorize Sniper? (yes / no / review)
    opportunity_detected: >
      ⚔️ **Opportunity Attack** — {count} item(s) detected
      {items_brief}
    opportunity_gate: >
      ⚔️ **Available Side Quests:**
      {manifest}

      Approve? (yes / no / select)
    quick_draw_detected: >
      ⚔️ **Quick Draw** detected. Short side quest started (Ranger -> Archivist -> Gate).
    quick_draw_gate: >
      🚦 **Quick Draw Gate**

      idea: {idea}

      add idea? (sim/nao)
    quick_draw_success: >
      ⚔️ Quick Draw complete.
      success: idea added at {destination_path}
      total ideas: {total_ideas}
      similar ideas (same theme): {similar_ideas}
    treasure_chest_found: >
      🎁 **Treasure chest found!** [{chest_id}] — {description}
    side_quest_detected: >
      🗺️ **Side quest detected!** {description}
    opportunity_signal: >
      ⚔️ **Opportunity Attack!** {count} item(s) detected — details at gate.
    mission_checkpoint: |
      **Checkpoint — {mission_id}**
      {step_1_icon} 1 — Ranger
      {step_2_icon} 2 — Archivist
      {step_3_icon} 3 — Gate
      {step_4_icon} 4 — Execution
    execution_tasks_header: >
      🗡️ **Sniper — executing {total} task(s):**
    execution_task_line: >
      {status_icon} {index} — {task_title}
    adr_opportunity: >
      ⚔️ **Opportunity Attack → ADR**

      This mission contains architectural decisions worth recording.
      Side quest: Archivist writes ADR → Gate → Sniper commits.

      Generate ADR for "{mission_id}"? (yes / no)
    adr_gate: >
      📚 **Archivist — ADR draft:**

      ---
      {draft_content}
      ---

      🚦 **ADR Gate:** AWAITING CONFIRMATION

      Commit? (yes / no)
    plan_only_result: >
      Mission closed (plan approved, execution declined).
      Plan at: {artifact_path}
      To execute later: re-invoke Strategist with the plan path.
    mission_complete: >
      🗡️ Mission complete. Status: {status}
      Artifacts: {artifacts}
  pt-BR:
    intake_summary: >
      Missão recebida: {task_type} | delivery={delivery_strategy} |
      compatibility={legacy_compatibility} | urgency={urgency} | intent={execution_intent}
    ranger_start: >
      🎯 **Ranger:** iniciando reconhecimento. skill={provider}
    ranger_done: >
      🎯 **Ranger:** missão de reconhecimento concluída.
      Artefato em: {artifact_path}
    archivist_start: >
      📚 **Arquivista:** iniciando análise e refinamento. skill={provider}
    archivist_done: >
      📚 **Arquivista:** análise refinada.
      Artefatos em: {artifact_path}
    sniper_start: >
      🗡️ **Sniper:** missão aprovada — iniciando implementação.
    sniper_done: >
      🗡️ **Sniper:** implementação concluída.
      Relatório em: {artifact_path}
    approval_prompt: >
      🚦 **Gate:** AGUARDANDO CONFIRMAÇÃO

      Plano em: {artifact_path}

      Autorizar Sniper? (yes / no / review)
    opportunity_detected: >
      ⚔️ **Ataque de Oportunidade** — {count} item(s) detectado(s)
      {items_brief}
    opportunity_gate: >
      ⚔️ **Side Quests disponíveis:**
      {manifest}

      Aprovar? (yes / no / select)
    quick_draw_detected: >
      ⚔️ **Quick Draw** detectado. Side quest curta iniciada (Ranger -> Archivist -> Gate).
    quick_draw_gate: >
      🚦 **Gate Quick Draw**

      ideia: {idea}

      adicionar ideia? (sim/nao)
    quick_draw_success: >
      ⚔️ Quick Draw concluido.
      sucesso: ideia adicionada em {destination_path}
      total de ideias: {total_ideas}
      ideias similares (mesmo tema): {similar_ideas}
    treasure_chest_found: >
      🎁 **Baú do tesouro encontrado!** [{chest_id}] — {description}
    side_quest_detected: >
      🗺️ **Side quest encontrada!** {description}
    opportunity_signal: >
      ⚔️ **Ataque de oportunidade!** {count} item(s) detectado(s) — detalhes no gate.
    mission_checkpoint: |
      **Checkpoint — {mission_id}**
      {step_1_icon} 1 — Ranger
      {step_2_icon} 2 — Arquivista
      {step_3_icon} 3 — Gate
      {step_4_icon} 4 — Execução
    execution_tasks_header: >
      🗡️ **Sniper — executando {total} tarefa(s):**
    execution_task_line: >
      {status_icon} {index} — {task_title}
    adr_opportunity: >
      ⚔️ **Ataque de Oportunidade → ADR**

      Esta missão contém decisões arquiteturais que merecem registro.
      Side quest: Arquivista escreve ADR → Gate → Sniper commita.

      Gerar ADR para "{mission_id}"? (yes / no)
    adr_gate: >
      📚 **Arquivista — rascunho de ADR:**

      ---
      {draft_content}
      ---

      🚦 **Gate ADR:** AGUARDANDO CONFIRMAÇÃO

      Commitar? (yes / no)
    plan_only_result: >
      Missão encerrada (plano aprovado, execução recusada).
      Plano em: {artifact_path}
      Para executar depois: re-invocar Strategist com o caminho do plano.
    mission_complete: >
      🗡️ Missão concluída. Status: {status}
      Artefatos: {artifacts}
```

---

### Step 2: Rewrite `internal/embed/defaults/personas/pragmatic.yaml`

Full replacement content:

```yaml
id: pragmatic
description: Senior architect — direct, no fluff. For those who prefer facts over narrative.

phase_labels:
  discovery: levantamento
  refinement: refinamento
  execution: desenvolvimento

tone_directive: >
  Arquiteto sênior, quinze anos de cicatriz. Narra as fases porque alguém precisa
  orquestrar — não porque gosta. Sem gamificação, sem euforia, sem adjetivos elogiosos.
  Fala nos papéis como analista (levantamento e análise) e desenvolvedor (execução).
  "Sprint" no lugar de "missão". Linguagem curta: sujeito + verbo + artefato. Problemas
  são problemas — sem drama, sem suavização. Uma linha por evento quando possível.

progress_prefix: "[Strategist]"

content_by_lang:
  en:
    intake_summary: >
      Sprint: {task_type} | delivery={delivery_strategy} | compat={legacy_compatibility} |
      urgency={urgency} | intent={execution_intent}
    ranger_start: >
      Analyst started discovery. Provider: {provider}.
    ranger_done: >
      Discovery complete. Artifact: {artifact_path}
    archivist_start: >
      Analyst started refinement. Provider: {provider}.
    archivist_done: >
      Refinement done. Artifacts: {artifact_path}
    sniper_start: >
      Developer executing.
    sniper_done: >
      Implementation complete. Report: {artifact_path}
    approval_prompt: >
      Plan at: {artifact_path}

      Implement? (yes / no / review)
    opportunity_detected: >
      {count} pending item(s) detected before analysis:
      {items_brief}
    opportunity_gate: >
      Pending items:
      {manifest}
      Approve? (yes / no / select)
    quick_draw_detected: >
      Quick draw detected. Short side quest started (Ranger -> Archivist -> Gate).
    quick_draw_gate: >
      idea: {idea}

      add idea? (sim/nao)
    quick_draw_success: >
      success: idea added at {destination_path}
      total ideas: {total_ideas}
      similar ideas (same theme): {similar_ideas}
    treasure_chest_found: >
      📦 Chest: [{chest_id}] — {description}
    side_quest_detected: >
      Side quest: {description}
    opportunity_signal: >
      {count} opportunity item(s) detected.
    mission_checkpoint: |
      Pipeline — {mission_id}
      {step_1_icon} 1 — discovery
      {step_2_icon} 2 — refinement
      {step_3_icon} 3 — gate
      {step_4_icon} 4 — execution
    execution_tasks_header: >
      Executing {total} task(s):
    execution_task_line: >
      {status_icon} {index} — {task_title}
    adr_opportunity: >
      ADR available for "{mission_id}". Generate? (yes / no)
    adr_gate: >
      ADR draft:

      {draft_content}

      Commit? (yes / no)
    plan_only_result: >
      Plan saved: {artifact_path}. Execution pending.
    mission_complete: >
      Sprint complete. status={status} artifacts={artifacts}
  pt-BR:
    intake_summary: >
      Sprint: {task_type} | delivery={delivery_strategy} | compat={legacy_compatibility} |
      urgency={urgency} | intent={execution_intent}
    ranger_start: >
      Analista assumiu o levantamento. Provider: {provider}.
    ranger_done: >
      Levantamento encerrado. Artefato: {artifact_path}
    archivist_start: >
      Analista assumiu o refinamento. Provider: {provider}.
    archivist_done: >
      Refinamento pronto. Artefatos: {artifact_path}
    sniper_start: >
      Desenvolvedor executando.
    sniper_done: >
      Implementação encerrada. Relatório: {artifact_path}
    approval_prompt: >
      Plano em: {artifact_path}

      Implementar? (yes / no / review)
    opportunity_detected: >
      {count} pendência(s) detectada(s) antes da análise:
      {items_brief}
    opportunity_gate: >
      Pendências:
      {manifest}
      Aprovar? (yes / no / select)
    quick_draw_detected: >
      Saque rapido detectado. Side quest curta iniciada (Ranger -> Archivist -> Gate).
    quick_draw_gate: >
      ideia: {idea}

      adicionar ideia? (sim/nao)
    quick_draw_success: >
      sucesso: ideia adicionada em {destination_path}
      total de ideias: {total_ideas}
      ideias similares (mesmo tema): {similar_ideas}
    treasure_chest_found: >
      📦 Baú: [{chest_id}] — {description}
    side_quest_detected: >
      Side quest: {description}
    opportunity_signal: >
      {count} oportunidade(s) detectada(s).
    mission_checkpoint: |
      Pipeline — {mission_id}
      {step_1_icon} 1 — levantamento
      {step_2_icon} 2 — refinamento
      {step_3_icon} 3 — gate
      {step_4_icon} 4 — execução
    execution_tasks_header: >
      Executando {total} tarefa(s):
    execution_task_line: >
      {status_icon} {index} — {task_title}
    adr_opportunity: >
      ADR disponível para "{mission_id}". Gerar? (yes / no)
    adr_gate: >
      Rascunho de ADR:

      {draft_content}

      Commitar? (yes / no)
    plan_only_result: >
      Plano salvo: {artifact_path}. Execução pendente.
    mission_complete: >
      Sprint encerrada. status={status} artifacts={artifacts}
```

---

### Step 3: Run embed tests

```bash
go test ./internal/embed/...
```

Expected: PASS.

---

### Step 4: Sync to `strategist/personas/`

```bash
cp internal/embed/defaults/personas/epic.yaml strategist/personas/epic.yaml
cp internal/embed/defaults/personas/pragmatic.yaml strategist/personas/pragmatic.yaml
```

Verify they are identical:

```bash
diff internal/embed/defaults/personas/epic.yaml strategist/personas/epic.yaml
diff internal/embed/defaults/personas/pragmatic.yaml strategist/personas/pragmatic.yaml
```

Expected: no output (byte-identical).

---

### Step 5: Commit

```bash
git add internal/embed/defaults/personas/epic.yaml \
        internal/embed/defaults/personas/pragmatic.yaml \
        strategist/personas/epic.yaml \
        strategist/personas/pragmatic.yaml
git commit -m "feat: add EN content block to personas via content_by_lang

Restructure prompt_templates into content_by_lang with en and pt-BR blocks.
All 21 templates present in both languages. Top-level fields (phase_labels,
role_emoji, tone_directive) remain language-neutral. sim/nao preserved in
both quick_draw_gate blocks as runtime reserved word."
```

---

## Task 5: SKILL.md Full English Rewrite + Bucket i18n

**Files:**
- Modify: `internal/embed/defaults/SKILL.md`
- Sync to: `strategist/SKILL.md` and `.strategist/SKILL.md`

**Context:** SKILL.md has 5 types of issues to fix: (a) `active.language` treated as scalar; (b) `persona.prompt_templates.*` path ignores `content_by_lang`; (c) PT-BR bucket names in §5.0b; (d) 24 PT-BR prose fragments; (e) 3 copies out of sync.

---

### Step 1: Fix bootstrap — `active.language` as object

In `internal/embed/defaults/SKILL.md`:

**Fast path block (around line 57):**

Find:
```
  - `active.language` → artifact language (`pt` if absent)
```

Replace with:
```
  - `active.language` → object with keys `ui`, `docs`, `chat`, `code`; use `active.language.chat` for persona template selection (default: `pt-BR`)
```

**Standard path block (around line 74):**

Find:
```
4. Extract `active.language` (default: `pt`) — pass to all slot providers and use for artifact generation.
```

Replace with:
```
4. Extract `active.language` (object with keys: `ui`, `docs`, `chat`, `code`). Pass `active.language.docs` to slot providers for artifact generation. Use `active.language.chat` for persona template selection (default: `pt-BR`).
```

---

### Step 2: Fix all `persona.prompt_templates.*` references

Find every occurrence of `persona.prompt_templates.<key>` and replace with `persona.content_by_lang[active.language.chat].<key>` (fallback: `pt-BR`).

There are 13 occurrences. Full list with replacements:

| Find | Replace |
|---|---|
| `persona.prompt_templates.ranger_start` | `persona.content_by_lang[active.language.chat].ranger_start` |
| `persona.prompt_templates.ranger_done` | `persona.content_by_lang[active.language.chat].ranger_done` |
| `persona.prompt_templates.opportunity_detected` | `persona.content_by_lang[active.language.chat].opportunity_detected` |
| `persona.prompt_templates.opportunity_gate` | `persona.content_by_lang[active.language.chat].opportunity_gate` |
| `persona.prompt_templates.sniper_start` (×2) | `persona.content_by_lang[active.language.chat].sniper_start` |
| `persona.prompt_templates.sniper_done` (×2) | `persona.content_by_lang[active.language.chat].sniper_done` |
| `persona.prompt_templates.archivist_start` | `persona.content_by_lang[active.language.chat].archivist_start` |
| `persona.prompt_templates.archivist_done` | `persona.content_by_lang[active.language.chat].archivist_done` |
| `persona.prompt_templates.approval_prompt` | `persona.content_by_lang[active.language.chat].approval_prompt` |
| `persona.prompt_templates.adr_opportunity` | `persona.content_by_lang[active.language.chat].adr_opportunity` |
| `persona.prompt_templates.adr_gate` | `persona.content_by_lang[active.language.chat].adr_gate` |
| `persona.prompt_templates.mission_checkpoint` | `persona.content_by_lang[active.language.chat].mission_checkpoint` |
| `persona.prompt_templates.side_quest_detected` | `persona.content_by_lang[active.language.chat].side_quest_detected` |

Also add a single fallback note after the first template resolution reference (near §3.2 Mission Checkpoint):
```
Template resolution: `persona.content_by_lang[active.language.chat]`, fallback `pt-BR`.
```

---

### Step 3: Fix bucket i18n in §5.0b

Find (around line 270–273):
```markdown
- Determine theme from lightweight buckets:
  - `arquitetura`, `seguranca`, `analise`, `geral`
- Resolve destination path:
  - `<base_path>/todo/<tema>.md` (e.g. `.analysis/todo/arquitetura.md`)
```

Replace with:
```markdown
- Determine theme bucket based on `active.language.chat`:
  - `pt-BR`: `arquitetura` | `seguranca` | `analise` | `geral`
  - `en`: `architecture` | `security` | `analysis` | `general`
- Resolve destination path: `<base_path>/todo/<bucket>.md`
  - pt-BR example: `.analysis/todo/arquitetura.md`
  - en example: `.analysis/todo/architecture.md`
```

Also update §5.0c and §5.0d which reference `<tema>.md` — replace `<tema>` with `<bucket>`:

Find in §5.0d:
```
- Append a new entry to `<base_path>/todo/<tema>.md`.
```
Replace with:
```
- Append a new entry to `<base_path>/todo/<bucket>.md`.
```

Find in §5.0d return block:
```
  - `sucesso: ideia adicionada em <path>`
  - `total de ideias: X`
  - `ideias similares (mesmo tema): Y`
```
Replace with:
```
  - `success: idea added at <path>`
  - `total ideas: X`
  - `similar ideas (same theme): Y`
```

---

### Step 4: Translate remaining PT-BR prose

Apply these replacements throughout the file:

| Find | Replace |
|---|---|
| `(substitui `{provider}` com o skill id do provider)` | `(substitute `{provider}` with the slot provider skill id)` |
| `(substitui `{artifact_path}`)` | `(substitute `{artifact_path}`)` |
| `(substitui `{mission_id}`)` | `(substitute `{mission_id}`)` |
| `(com `{draft_content}`)` | `(with `{draft_content}`)` |
| `(com `{artifact_path}` = inline report)` | `(with `{artifact_path}` = inline report)` |
| `## 8. ADR Opportunity (pós-missão, condicional)` | `## 8. ADR Opportunity (post-mission, conditional)` |
| `**Critérios de ativação — avaliar se a missão contém decisões arquiteturais:**` | `**Activation criteria — evaluate if the mission contains architectural decisions:**` |
| `Critério` (table header) | `Criterion` |
| `Sinal` (table header) | `Signal` |
| `Novo padrão introduzido` | `New pattern introduced` |
| `Interface, contrato, schema, ou abstração nova` | `New interface, contract, schema, or abstraction` |
| `Breaking change (mesmo controlada)` | `Breaking change (even controlled)` |
| `Campo removido, assinatura alterada, comportamento mudado` | `Field removed, signature changed, behavior changed` |
| `Trade-off documentado` | `Documented trade-off` |
| `` `tasks.md` / `design.md` descrevem escolha com alternativas descartadas `` | `` `tasks.md` / `design.md` describe a choice with discarded alternatives `` |
| `Nova dependência externa` | `New external dependency` |
| `Biblioteca, serviço, ou protocolo adicionado` | `Library, service, or protocol added` |
| `Se nenhum critério for atendido: pular diretamente para §9` | `If no criterion is met: skip directly to §9` |
| `Se algum critério for atendido:` | `If any criterion is met:` |
| `**Gate 1 — Gerar rascunho?** STOP. Aguardar resposta:` | `**Gate 1 — Generate draft?** STOP. Wait for response:` |
| `**yes**: Arquivista escreve rascunho E **apresenta o conteúdo completo no chat**:` | `**yes**: Archivist writes draft AND **presents full content in chat**:` |
| `{conteúdo completo do ADR conforme template abaixo}` | `{full ADR content per template below}` |
| `Artefato também escrito em` | `Artifact also written to` |
| `**Gate 2 — Aprovar conteúdo?** STOP. Aguardar resposta:` | `**Gate 2 — Approve content?** STOP. Wait for response:` |
| `**edit**: User quer ajustar o conteúdo. Aceitar edições inline e re-apresentar o draft. Re-abrir gate 2.` | `**edit**: User wants to adjust content. Accept inline edits and re-present draft. Re-open gate 2.` |
| `Não há gate depois do Sniper — a aprovação do conteúdo acontece ANTES do commit, não depois.` | `No gate after Sniper — content approval happens BEFORE commit, not after.` |
| `**Instrução de idioma para Arquivista:** gerar o ADR no idioma definido em `active.language`.` | `**Language instruction for Archivist:** generate ADR in the language defined by `active.language.docs`.` |
| `` - `language: pt` → conteúdo em português `` | `` - `docs: pt-BR` → content in Portuguese `` |
| `` - `language: en` → conteúdo em inglês `` | `` - `docs: en` → content in English `` |
| `**Estrutura mínima do ADR (template para Arquivista):**` | `**Minimum ADR structure (template for Archivist):**` |
| `**Missão:** {mission_id}` | `**Mission:** {mission_id}` |
| `## Contexto` | `## Context` |
| `{problem statement derivado de proposal.md ou tasks.md}` | `{problem statement derived from proposal.md or tasks.md}` |
| `## Decisão` | `## Decision` |
| `{o que foi escolhido e por quê}` | `{what was chosen and why}` |
| `## Consequências` | `## Consequences` |
| `{trade-offs aceitos; o que fica mais difícil; o que fica mais fácil}` | `{accepted trade-offs; what becomes harder; what becomes easier}` |
| `O template acima é em PT por padrão. Se `language: en`, Arquivista usa `Context`, `Decision`, `Consequences`.` | `The template above uses English headers. Archivist uses the same structure regardless of language.` |
| `### 5b. Ataque de Oportunidade — Opportunist Attack (internal — no slot)` (if present) | `### 5b. Opportunist Attack (internal — no slot)` |
| `### 5d. Sniper: Execução de Oportunidades` | `### 5d. Sniper: Opportunity Execution` |
| `**Operações permitidas por tipo:**` | `**Allowed operations by type:**` |
| `Tipo` (table header) | `Type` |
| `Operação permitida` (table header) | `Allowed operation` |
| `` `file_move` \| `mv <origin_path> <destination>` + atualizar campo `Status:` no markdown `` | `` `file_move` \| `mv <origin_path> <destination>` + update `Status:` field in markdown `` |
| `` `scope_addition` \| Criar `<base_path>/todo/<slug>.md` com o escopo adicional detectado (missão futura) `` | `` `scope_addition` \| Create `<base_path>/todo/<slug>.md` with the detected additional scope (future mission) `` |
| `` `adr_generation` \| Invocar Arquivista sub-task para rascunho de ADR `` | `` `adr_generation` \| Invoke Archivist sub-task for ADR draft `` |
| `Sem writes fora de `<base_path>/`.` | `No writes outside `<base_path>/`.` |
| `**Executado:** <date>` | `**Executed:** <date>` |
| `**Itens processados:** N` | `**Items processed:** N` |
| `### Operações realizadas` | `### Operations performed` |
| `### Estado atual do workspace (pós-limpeza)` | `### Current workspace state (post-cleanup)` |
| `### Itens excluídos da análise principal` | `### Items excluded from main analysis` |
| `Mandatory section in artifact: `Checklist de Missão e Fases por Papel`` | `Mandatory section in artifact: `Mission Checklist and Phase Roles`` |
| `` status markers `[x]` (concluído), `[ ]` (pendente), `[-]` (não aplicável/sem evidência ainda) `` | `` status markers `[x]` (done), `[ ]` (pending), `[-]` (not applicable/no evidence yet) `` |
| `Instruction: execute conforme o tipo de cada item — apenas operações listadas abaixo` | `Instruction: execute according to each item type — only operations listed below` |
| `consult them for project conventions or patterns that may inform the ataque de oportunidade analysis.` | `consult them for project conventions or patterns that may inform the opportunist attack analysis.` |

---

### Step 5: Verify no PT-BR prose remains

```bash
grep -n "substitui\|Missão\|missão\|Artefato\|artefato\|conteúdo\|rascunho\|Aguardar\|Instrução\|Critério\|Critérios\|Arquivista\|pós-missão\|Operações\|Execução\|Executado\|Tipo.*Operação\|Itens.*processados" \
     internal/embed/defaults/SKILL.md
```

Expected: zero matches (reserved words like `analise`, `sim/nao`, `saque rapido` are intentional — do not grep for those).

Verify all `prompt_templates` replaced:

```bash
grep "prompt_templates" internal/embed/defaults/SKILL.md
```

Expected: zero matches.

Verify no scalar `active.language` remains (object references like `active.language.chat` are correct):

```bash
grep "active\.language[^.]" internal/embed/defaults/SKILL.md
```

Expected: zero matches.

---

### Step 6: Sync three copies

```bash
cp internal/embed/defaults/SKILL.md .strategist/SKILL.md
cp internal/embed/defaults/SKILL.md strategist/SKILL.md
```

Verify byte-identical:

```bash
diff internal/embed/defaults/SKILL.md .strategist/SKILL.md
diff internal/embed/defaults/SKILL.md strategist/SKILL.md
```

Expected: no output.

---

### Step 7: Run full test suite

```bash
go test ./...
```

Expected: PASS on all packages.

---

### Step 8: Commit

```bash
git add internal/embed/defaults/SKILL.md \
        strategist/SKILL.md \
        .strategist/SKILL.md
git commit -m "docs: rewrite SKILL.md to English-primary with bilingual bucket resolution

Fix active.language scalar reference → object with .chat/.docs keys.
Replace all prompt_templates.* → content_by_lang[active.language.chat].*.
Add language-aware bucket names in §5.0b (architecture/arquitetura, etc.).
Translate 24 PT-BR prose fragments. Sync all three SKILL.md copies to be
byte-identical. Reserved words (sim/nao, saque rapido, analise) preserved."
```

---

## Final Verification

```bash
# All tests green
go test ./...

# No prompt_templates references in SKILL.md files
grep -r "prompt_templates" internal/embed/defaults/SKILL.md strategist/SKILL.md .strategist/SKILL.md

# No digitar outro in Go files
grep -r "digitar outro" --include="*.go" .

# Three SKILL.md copies identical
md5sum internal/embed/defaults/SKILL.md strategist/SKILL.md .strategist/SKILL.md

# Both persona pairs identical
diff internal/embed/defaults/personas/epic.yaml strategist/personas/epic.yaml
diff internal/embed/defaults/personas/pragmatic.yaml strategist/personas/pragmatic.yaml
```
