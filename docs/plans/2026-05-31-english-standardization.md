# English Standardization and i18n Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Standardize all internal Go identifiers and agent instruction documents to English, extract the existing wizard i18n bundle into a proper package, and add multilanguage support to persona templates via `content_by_lang`.

**Architecture:** Four independent tasks executed in sequence. Tasks 1 and 2 are pure Go refactors with full test coverage. Task 3 is a YAML restructure of embedded defaults. Task 4 is a prose rewrite of SKILL.md with no behaviour change.

**Tech Stack:** Go 1.22+, `go test ./...`, YAML, Markdown. No new external dependencies.

---

## Task 1: Rename PT-BR Go Constants to English

Rename seven PT-BR constant identifiers in `internal/domain/policy.go`. String values (reserved words) do not change.

**Files:**
- Modify: `internal/domain/policy.go`
- Modify: `internal/domain/state_machine.go`
- Modify: `internal/domain/state_machine_test.go`
- Modify: `internal/domain/policy_evaluator_test.go`

### Step 1: Rename constants in policy.go

In `internal/domain/policy.go`, apply these renames — identifier names only, string values stay:

```go
// Before → After (values unchanged)
DoneScopeAnalise  → DoneScopeAnalysis   // "analise"
DoneScopeEntrega  → DoneScopeDelivery   // "entrega"

MissionModeAnalise          → MissionModeAnalysis         // "analise"
MissionModeEntregaRevisada  → MissionModeRevisedDelivery  // "entrega_revisada"
MissionModeEntregaExecutada → MissionModeExecutedDelivery // "entrega_executada"

StateDoneAnalise → StateDoneAnalysis   // MissionState = "DONE_ANALISE"
StateDoneEntrega → StateDoneDelivery   // MissionState = "DONE_ENTREGA"
```

All occurrences in policy.go:
- const block declarations (lines 5–6, 11–13, 34–35)
- body of `MissionModeFromLegacy` — references `DoneScopeAnalise`, `MissionModeAnalise`, `MissionModeEntregaExecutada`, `MissionModeEntregaRevisada`
- body of `NewMissionPolicy` — switch cases and struct literals reference all five MissionMode and DoneScope constants
- body of `NormalizePolicy` — references `MissionModeEntregaExecutada`

Also update the comment on `MissionPolicy.Mode` field (line 56) from:
```go
Mode string // analise | entrega_revisada | entrega_executada
```
to:
```go
Mode string // reserved words: "analise" | "entrega_revisada" | "entrega_executada"
```

And `MissionPolicy.ExpectsDelivery` comment (line 58) from:
```go
ExpectsDelivery string // analise | entrega
```
to:
```go
ExpectsDelivery string // reserved words: "analise" | "entrega"
```

### Step 2: Update state_machine.go

In `internal/domain/state_machine.go`, replace all references:
- `StateDoneAnalise` → `StateDoneAnalysis` (lines 22, 23, 83, 86, 98)
- `StateDoneEntrega` → `StateDoneDelivery` (lines 24, 25, 103, 113)
- `MissionModeAnalise` → `MissionModeAnalysis` (line 85)

### Step 3: Update test files

In `internal/domain/state_machine_test.go`, replace:
- `domain.MissionModeAnalise` → `domain.MissionModeAnalysis` (lines 13, 30, 55)
- `domain.MissionModeEntregaRevisada` → `domain.MissionModeRevisedDelivery` (lines 30, 57)
- `domain.MissionModeEntregaExecutada` → `domain.MissionModeExecutedDelivery` (line 53)

In `internal/domain/policy_evaluator_test.go`, replace:
- `domain.DoneScopeEntrega` → `domain.DoneScopeDelivery` (lines 14, 54, 81)
- `domain.DoneScopeAnalise` → `domain.DoneScopeAnalysis` (lines 41, 67)
- `domain.MissionModeEntregaExecutada` → `domain.MissionModeExecutedDelivery` (lines 34, 108)
- `domain.MissionModeAnalise` → `domain.MissionModeAnalysis` (line 95)

### Step 4: Verify compilation and tests

```bash
go build ./...
go test ./internal/domain/...
```

Expected: all tests pass, zero compilation errors.

### Step 5: Update types.go comment

In `internal/domain/types.go` lines 64–66, update the field comments to reference the reserved word nature:

```go
MissionMode        string // reserved word: "analise" | "entrega_revisada" | "entrega_executada"
DoneScope          string // reserved word: "analise" | "entrega"
ApplyChanges       bool   // false by default; forced false when DoneScope="analise"
```

### Step 6: Update protocol.md reference

In `internal/embed/defaults/protocol.md` line 45, the text references `escopo_done`/`aplicar_alteracoes` — update to note these are legacy YAML fields:

Find: `with legacy \`escopo_done\`/\`aplicar_alteracoes\` derivation`
Keep as-is — these are YAML field names (reserved words), not Go identifiers. No change needed here.

### Step 7: Run full test suite and commit

```bash
go test ./...
```

Expected output:
```
ok  github.com/SergioLacerda/strategist-skill/cmd/strategist
ok  github.com/SergioLacerda/strategist-skill/internal/compile
ok  github.com/SergioLacerda/strategist-skill/internal/domain
ok  github.com/SergioLacerda/strategist-skill/internal/embed
ok  github.com/SergioLacerda/strategist-skill/internal/install
ok  github.com/SergioLacerda/strategist-skill/internal/stale
ok  github.com/SergioLacerda/strategist-skill/internal/telemetry
ok  github.com/SergioLacerda/strategist-skill/strategist/tests
```

```bash
git add internal/domain/
git commit -m "refactor: rename PT-BR Go constant identifiers to English

Reserved word string values (analise, entrega, entrega_revisada,
entrega_executada) are unchanged — they are intentional PT-BR
identifiers matched by the agent at runtime."
```

---

## Task 2: Extract i18n Package from Wizard

The wizard already has `wizardStrings`, `bundleEN`, `bundlePTBR`, and `bundleFor()` inside `internal/install/wizard.go`. Extract them into a dedicated `internal/i18n/` package with exported types, and fix the one gap: `customLabel` is hardcoded outside the bundle.

**Files:**
- Create: `internal/i18n/wizard.go`
- Create: `internal/i18n/wizard_test.go`
- Modify: `internal/install/wizard.go`

### Step 1: Write the failing test first

Create `internal/i18n/wizard_test.go`:

```go
package i18n_test

import (
	"reflect"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

func TestBundleFor_routing(t *testing.T) {
	if got := i18n.BundleFor("pt-BR"); got != i18n.PT {
		t.Errorf("BundleFor(pt-BR): got %v, want PT bundle", got)
	}
	if got := i18n.BundleFor("en"); got != i18n.EN {
		t.Errorf("BundleFor(en): got %v, want EN bundle", got)
	}
	// Unknown locale defaults to EN
	if got := i18n.BundleFor("fr"); got != i18n.EN {
		t.Errorf("BundleFor(fr): got %v, want EN bundle (default)", got)
	}
}

func TestBundles_completeness(t *testing.T) {
	bundles := map[string]i18n.WizardStrings{
		"EN":    i18n.EN,
		"PT":    i18n.PT,
	}
	for name, b := range bundles {
		v := reflect.ValueOf(b)
		typ := v.Type()
		for i := range v.NumField() {
			field := typ.Field(i)
			if v.Field(i).String() == "" {
				t.Errorf("bundle %s: field %s is empty", name, field.Name)
			}
		}
	}
}
```

### Step 2: Run test to confirm it fails

```bash
go test ./internal/i18n/...
```

Expected: FAIL — package does not exist yet.

### Step 3: Create internal/i18n/wizard.go

```go
// Package i18n provides localised string bundles for user-facing CLI surfaces.
package i18n

import "strings"

// WizardStrings holds all user-visible strings for the install wizard.
type WizardStrings struct {
	PromptDocLang    string
	PromptChatLang   string
	PromptCodeLang   string
	PromptMode       string
	PromptMissionMode string
	PromptBasePath   string
	PromptAdr        string
	HeaderSlots      string
	PromptDiscovery  string
	PromptRefinement string
	PromptExecution  string
	HeaderChest      string
	PromptChestPath  string
	LabelCustomInput string
}

// EN is the English wizard bundle.
var EN = WizardStrings{
	PromptDocLang:    "Documentation language",
	PromptChatLang:   "Chat/interaction language",
	PromptCodeLang:   "Code language",
	PromptMode:       "Mode",
	PromptMissionMode: "DONE scope (analysis-only or analysis + implementation)\n  analise = analysis-only\n  entrega_revisada = analysis + handoff (no implementation)\n  entrega_executada = analysis + implementation",
	PromptBasePath:   "Base path for analysis workspace",
	PromptAdr:        "Enable ADR generation at mission end?",
	HeaderSlots:      "\nSlot providers — which skill fills each mission role:",
	PromptDiscovery:  "  Ranger / discovery provider",
	PromptRefinement: "  Arquivista / refinement provider",
	PromptExecution:  "  Sniper / execution provider",
	HeaderChest:      "\nTreasure chest — optional offline knowledge source for all slots:",
	PromptChestPath:  "  Knowledge source path (e.g. .sdd/source)",
	LabelCustomInput: "(enter other...)",
}

// PT is the PT-BR wizard bundle.
var PT = WizardStrings{
	PromptDocLang:    "Idioma da documentação",
	PromptChatLang:   "Idioma do chat/interação",
	PromptCodeLang:   "Idioma do código",
	PromptMode:       "Modo",
	PromptMissionMode: "Escopo do DONE (apenas análise ou análise + implementação)\n  analise = apenas análise\n  entrega_revisada = análise + handoff (sem implementação)\n  entrega_executada = análise + implementação",
	PromptBasePath:   "Caminho base do workspace de análise",
	PromptAdr:        "Habilitar geração de ADR ao final da missão?",
	HeaderSlots:      "\nProvedores de slot — qual skill preenche cada papel da missão:",
	PromptDiscovery:  "  Ranger / provedor de descoberta",
	PromptRefinement: "  Arquivista / provedor de refinamento",
	PromptExecution:  "  Sniper / provedor de execução",
	HeaderChest:      "\nBaú do tesouro — base de conhecimento offline opcional para todos os slots:",
	PromptChestPath:  "  Caminho da base de conhecimento (ex: .sdd/source)",
	LabelCustomInput: "(digitar outro...)",
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

### Step 4: Run tests to confirm they pass

```bash
go test ./internal/i18n/...
```

Expected: PASS.

### Step 5: Update wizard.go to use the i18n package

In `internal/install/wizard.go`:

**Remove** the following (lines 10–63):
```go
type wizardStrings struct { ... }
var bundleEN = wizardStrings{ ... }
var bundlePTBR = wizardStrings{ ... }
func bundleFor(lang string) wizardStrings { ... }
```

**Add** import:
```go
"github.com/SergioLacerda/strategist-skill/internal/i18n"
```

**Replace** in `runWizard`:
```go
// Before
b := bundleFor(uiLang)
// After
b := i18n.BundleFor(uiLang)
```

**Replace** hardcoded customLabel (line 123):
```go
// Before
customLabel := "(digitar outro...)"
// After — use b.LabelCustomInput in each SelectOrInput call directly
```

The three `p.SelectOrInput(...)` calls pass `customLabel` as the last argument. Replace `customLabel` with `b.LabelCustomInput` in all three calls:

```go
discovery, err := p.SelectOrInput(b.PromptDiscovery, "brainstorming", []string{"brainstorming"}, b.LabelCustomInput)
refinement, err := p.SelectOrInput(b.PromptRefinement, "openspec-explore", []string{"openspec-explore"}, b.LabelCustomInput)
execution, err := p.SelectOrInput(b.PromptExecution, "sdd-ask", []string{"sdd-ask", "sdd-ask-full"}, b.LabelCustomInput)
```

Remove the now-unused `customLabel` variable declaration entirely.

Also remove the now-unused `normLang` function only if it is not called elsewhere — verify with `grep -n normLang internal/install/wizard.go`. It is called on lines 74, 82, 88, 94 — keep it.

### Step 6: Verify compilation and tests

```bash
go build ./...
go test ./internal/install/... ./internal/i18n/...
```

Expected: all pass. No user-visible string literals remain in wizard.go (verify with: `grep -n '"[A-Z]' internal/install/wizard.go` — should return empty).

### Step 7: Commit

```bash
git add internal/i18n/ internal/install/wizard.go
git commit -m "refactor: extract wizard i18n bundle to internal/i18n package

Moves wizardStrings struct and EN/PT-BR bundles from install package
to a dedicated i18n package with exported types. Adds LabelCustomInput
field to cover the previously hardcoded PT-BR string in wizard.go."
```

---

## Task 3: Add content_by_lang to Personas

Restructure `epic.yaml` and `pragmatic.yaml` so prompt templates are keyed by language under `content_by_lang`. The agent selects `content_by_lang[active.language.chat]` at runtime, falling back to `pt-BR`.

**Files:**
- Modify: `internal/embed/defaults/personas/epic.yaml`
- Modify: `internal/embed/defaults/personas/pragmatic.yaml`
- Modify: `.strategist/SKILL.md` — update persona loading instruction (done in Task 4)
- Modify: `internal/embed/defaults/SKILL.md` — update persona loading instruction (done in Task 4)
- Modify: `strategist/SKILL.md` — update persona loading instruction (done in Task 4)

### Step 1: Restructure epic.yaml

Replace the flat `prompt_templates:` block with `content_by_lang:`. Fields that stay at top level: `id`, `description`, `phase_labels`, `tone_directive`, `progress_prefix`, `role_emoji`.

The full restructured `internal/embed/defaults/personas/epic.yaml`:

```yaml
id: epic
description: Narrador épico — arquiteto que vê cada sprint como uma aventura. Para missões que merecem uma história.

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

### Step 2: Restructure pragmatic.yaml

Same structural change. Top-level fields: `id`, `description`, `phase_labels`, `tone_directive`, `progress_prefix`.

The full restructured `internal/embed/defaults/personas/pragmatic.yaml`:

```yaml
id: pragmatic
description: Arquiteto sênior — direto, sem enrolação. Para quem prefere fatos a narrativa.

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
      Analyst started survey. Provider: {provider}.
    ranger_done: >
      Survey done. Artifact: {artifact_path}
    archivist_start: >
      Analyst started refinement. Provider: {provider}.
    archivist_done: >
      Refinement done. Artifacts: {artifact_path}
    sniper_start: >
      Developer executing.
    sniper_done: >
      Implementation done. Report: {artifact_path}
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
      Quick Draw detected. Short side quest started (Ranger -> Archivist -> Gate).
    quick_draw_gate: >
      idea: {idea}

      add idea? (sim/nao)
    quick_draw_success: >
      success: idea added at {destination_path}
      total ideas: {total_ideas}
      similar ideas (same theme): {similar_ideas}
    adr_opportunity: >
      ADR available for "{mission_id}". Generate? (yes / no)
    adr_gate: >
      ADR draft:

      {draft_content}

      Commit? (yes / no)
    plan_only_result: >
      Plan saved: {artifact_path}. Execution pending.
    mission_complete: >
      Sprint done. status={status} artifacts={artifacts}
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

### Step 3: Verify embed tests still pass

The embed package tests validate that default files load correctly.

```bash
go test ./internal/embed/...
```

Expected: PASS.

### Step 4: Commit

```bash
git add internal/embed/defaults/personas/
git commit -m "feat: add content_by_lang blocks to persona templates

Restructures epic and pragmatic personas to support multilanguage
agent chat responses. Selector: active.language.chat, fallback pt-BR.
PT-BR content preserved verbatim; EN translations added."
```

---

## Task 4: Rewrite SKILL.md to English-Primary

Translate all instructional prose in SKILL.md to English. No protocol or behaviour changes — only the language of instructions. PT-BR reserved words (trigger phrases, config values) are preserved.

**Files:**
- Modify: `internal/embed/defaults/SKILL.md`
- Modify: `.strategist/SKILL.md`
- Modify: `strategist/SKILL.md`

**Preserved PT-BR reserved words (do not translate):**
- `saque rapido` and `quick draw` — user-typed trigger phrases (§3.1)
- `sim/nao` — quick draw gate responses (§5.0c)
- `"analise"`, `"entrega_revisada"`, `"entrega_executada"` — mission mode config values
- Event format strings: `[Strategist] phase=...` (these are code, not prose)

### Step 1: Translate internal/embed/defaults/SKILL.md

Apply all translations below to `internal/embed/defaults/SKILL.md`. Work section by section.

**Section headers to translate:**

| Before | After |
|---|---|
| `## 3.1 Quick Draw Route (Saque Rapido)` | `## 3.1 Quick Draw Route` |
| `### 5.0 Quick Draw Side Quest (conditional)` | `### 5.0 Quick Draw Side Quest (conditional)` ← already EN |
| `#### 5.0a Ranger (quick_draw)` | already EN |
| `#### 5.0b Archivist (quick_draw)` | already EN |
| `#### 5.0c Quick Draw Gate (mandatory)` | already EN |
| `#### 5.0d Sniper (quick_draw append)` | already EN |
| `### 5.0c Quick Draw Gate (mandatory)` | already EN |
| `### 5c. Gate de Oportunidade (conditional ...` | `### 5c. Opportunity Gate (conditional — only if opportunity manifest is non-empty)` |
| `### 5d. Sniper: Execução de Oportunidades (conditional ...` | `### 5d. Sniper: Opportunity Execution (conditional — only if opportunity gate approved)` |
| `## 8. ADR Opportunity (pós-missão, condicional)` | `## 8. ADR Opportunity (post-mission, conditional)` |

**Section 3.1 body** — translate the note. Current:
```
If the user explicitly requests quick capture (examples: `quick draw`, `saque rapido`,
`TODO` as rapid note), route to a dedicated side-quest flow.
```
Keep as-is — already in English with PT-BR preserved as trigger phrase.

**Section 5.0b** — translate bucket list and path note:

Current:
```
- Determine theme from lightweight buckets:
  - `arquitetura`, `seguranca`, `analise`, `geral`
- Resolve destination path:
  - `<base_path>/todo/<tema>.md` (e.g. `.analysis/todo/arquitetura.md`)
- Inspect existing file content (if present) and compute:
  - `total_ideas`: total idea entries in the destination theme file
  - `similar_ideas`: ideas in the same theme with textual similarity to the normalized idea
```
These bucket names (`arquitetura`, `seguranca`, `analise`, `geral`) are reserved words — keep them. The surrounding prose is already in English. No change needed here.

**Section 5.0c Quick Draw Gate** — the gate output block uses `ideia:` and `adicionar ideia? (sim/nao)`. These are reserved words — keep them. The instruction prose around it:

Current:
```
STOP. Show exactly:
```text
ideia: <texto_normalizado>
adicionar ideia? (sim/nao)
```
Wait for response:
- `sim`: proceed to Sniper append.
- `nao`: return without writing.
```

Keep `ideia:`, `(sim/nao)`, `sim`, `nao` as reserved words. The prose is already in English. No change needed.

**Section 5.0d** — translate the return block. Current:
```
- Return:
  - `sucesso: ideia adicionada em <path>`
  - `total de ideias: X`
  - `ideias similares (mesmo tema): Y`
```
Replace with:
```
- Return:
  - `sucesso: ideia adicionada em <path>`  ← persona template reserved word
  - `total de ideias: X`  ← persona template reserved word
  - `ideias similares (mesmo tema): Y`  ← persona template reserved word
```
Note: these match the persona template keys — preserve them as reserved words with a note. Actually, the persona now has both `en` and `pt-BR` blocks, so the agent uses the right one. The SKILL.md instruction here is about what Sniper returns — it should describe the template key, not the actual string. Update to:
```
- Return the `quick_draw_success` persona template result, substituting:
  - `{destination_path}` — path where the idea was appended
  - `{total_ideas}` — total idea count in that theme file
  - `{similar_ideas}` — count of ideas with textual similarity in the same theme
```

**Section 5a** — translate inline parenthetical. Current:
```
Emit via `persona.prompt_templates.ranger_start` (substitui `{provider}` com o skill id do provider).
```
Replace with:
```
Emit via `persona.prompt_templates.ranger_start` (substitute `{provider}` with the slot provider skill id).
```

Find and replace all occurrences of `(substitui ...)` pattern:
- `(substitui \`{provider}\` com o skill id do provider)` → `(substitute \`{provider}\` with the slot provider skill id)`
- `(substitui \`{artifact_path}\`)` → `(substitute \`{artifact_path}\`)`
- `(com \`{artifact_path}\` = inline report)` → `(\`{artifact_path}\` = inline report)`

**Section 5.0a** — translate body. Current:
```
- Output: one normalized line, preserving context:
  - `ideia: <formalizacao sem expandir escopo>`
- Ranger must not add requirements, milestones, or implementation details.
```
`ideia:` is a reserved word — keep it. The prose is already in English.

**Section 5.0b bucket note** — current text references `<tema>` in the path. Keep — it is a variable placeholder, not prose.

**Section 5b Opportunist Attack** — check for PT-BR prose. Current table:
```
| Directory | Check | Side quest type |
| `todo/` | Does this spec have a corresponding implementation commit in git? | `file_move` |
```
Already in English. Check the note:
```
**Heuristic for `file_move`:** git log contains a commit referencing the spec slug ... When uncertain, list as a candidate — the user decides at the gate.
```
Already in English.

Emit note: `{items_brief}` line:
```
- `{items_brief}` = one line per item: `→ <slug> reason: <motivo>`
```
Replace `<motivo>` with `<reason>`:
```
- `{items_brief}` = one line per item: `→ <slug> reason: <reason>`
```

**Section 5c Opportunity Gate** — the manifest block uses `Motivo:`. Replace:
```
     Motivo: <reason>
```
with:
```
     Reason: <reason>
```

**Section 5d Sniper Opportunity Execution** — translate the instruction and table. Current:
```
- Instruction: execute conforme o tipo de cada item — apenas operações listadas abaixo

**Operações permitidas por tipo:**

| Tipo | Operação permitida |
|------|--------------------|
| `file_move` | `mv <origin_path> <destination>` + atualizar campo `Status:` no markdown |
| `scope_addition` | Criar `<base_path>/todo/<slug>.md` com o escopo adicional detectado (missão futura) |
| `adr_generation` | Invocar Arquivista sub-task para rascunho de ADR em `<base_path>/done/<mission_id>-adr.md` |

Sem writes fora de `<base_path>/`.
```
Replace with:
```
- Instruction: execute each item according to its type — only the operations listed below.

**Allowed operations by type:**

| Type | Allowed operation |
|------|-------------------|
| `file_move` | `mv <origin_path> <destination>` + update `Status:` field in the markdown |
| `scope_addition` | Create `<base_path>/todo/<slug>.md` with the detected additional scope (future mission) |
| `adr_generation` | Invoke Archivist sub-task to draft ADR at `<base_path>/done/<mission_id>-adr.md` |

No writes outside `<base_path>/`.
```

**Section 5d opportunity report block** — translate the inline markdown template. Current:
```markdown
## Opportunity Report
**Executado:** <date> | **Itens processados:** N

### Operações realizadas
- `<origin>` → `<destination>` (file_move)
- `<slug>.md` criado em todo/ (scope_addition)

### Estado atual do workspace (pós-limpeza)
- `todo/`: N itens
- `pending/`: N itens
- `refined/`: N itens
- `done/`: N itens

### Itens excluídos da análise principal
<list — Archivist must not treat these as pending work>
```
Replace with:
```markdown
## Opportunity Report
**Executed:** <date> | **Items processed:** N

### Operations performed
- `<origin>` → `<destination>` (file_move)
- `<slug>.md` created in todo/ (scope_addition)

### Current workspace state (post-cleanup)
- `todo/`: N items
- `pending/`: N items
- `refined/`: N items
- `done/`: N items

### Items excluded from main analysis
<list — Archivist must not treat these as pending work>
```

**Section 5e Archivist** — translate injection instruction. Current:
```
> "Items listed under 'Itens excluídos da análise principal' are resolved. Do not treat them as pending. Base your analysis on the post-cleanup workspace state."
```
Replace with:
```
> "Items listed under 'Items excluded from main analysis' are resolved. Do not treat them as pending. Base your analysis on the post-cleanup workspace state."
```

**Section 5.0b** — translate `tema` comment in path:
```
- `<base_path>/todo/<tema>.md` (e.g. `.analysis/todo/arquitetura.md`)
```
Keep `<tema>` and `arquitetura` as reserved words — they are bucket names, not prose.

**Section 5 mandatory sweep invariant** — current:
```
"Foco em alvo único" is NOT a valid reason to skip sweeps.
```
Replace with:
```
"Single-target focus" is NOT a valid reason to skip sweeps.
```

**Section 8 ADR** — translate all PT-BR prose.

Current activation criteria table:
```
**Critérios de ativação — avaliar se a missão contém decisões arquiteturais:**

| Critério | Sinal |
|----------|-------|
| Novo padrão introduzido | Interface, contrato, schema, ou abstração nova |
| Breaking change (mesmo controlada) | Campo removido, assinatura alterada, comportamento mudado |
| Trade-off documentado | `tasks.md` / `design.md` descrevem escolha com alternativas descartadas |
| Nova dependência externa | Biblioteca, serviço, ou protocolo adicionado |

Se nenhum critério for atendido: pular diretamente para §9 (Learning Phase).

Se algum critério for atendido:
```
Replace with:
```
**Activation criteria — evaluate whether the mission contains architectural decisions:**

| Criterion | Signal |
|-----------|--------|
| New pattern introduced | New interface, contract, schema, or abstraction |
| Breaking change (even controlled) | Field removed, signature changed, behaviour changed |
| Documented trade-off | `tasks.md` / `design.md` describe a choice with discarded alternatives |
| New external dependency | Library, service, or protocol added |

If no criterion is met: skip directly to §9 (Learning Phase).

If any criterion is met:
```

Current gate 1 block:
```
**Gate 1 — Gerar rascunho?** STOP. Aguardar resposta:
- **no**: Registrar na learning phase como "ADR recusado (gate 1)". Continuar para §9.
- **yes**: Arquivista escreve rascunho E **apresenta o conteúdo completo no chat**:
```
Replace with:
```
**Gate 1 — Generate draft?** STOP. Wait for response:
- **no**: Record in learning phase as "ADR declined (gate 1)". Continue to §9.
- **yes**: Archivist writes draft AND **presents the full content in chat**:
```

Current gate 2 block:
```
  **Gate 2 — Aprovar conteúdo?** STOP. Aguardar resposta:
  - **yes**: Sniper commita o ADR. `mission_result.adr = <path>`. Continuar para §9.
  - **no**: ADR descartado (arquivo removido). `mission_result.status = completed` (sem ADR). Continuar para §9.
  - **edit**: User quer ajustar o conteúdo. Aceitar edições inline e re-apresentar o draft. Re-abrir gate 2.

Não há gate depois do Sniper — a aprovação do conteúdo acontece ANTES do commit, não depois.
```
Replace with:
```
  **Gate 2 — Approve content?** STOP. Wait for response:
  - **yes**: Sniper commits the ADR. `mission_result.adr = <path>`. Continue to §9.
  - **no**: ADR discarded (file removed). `mission_result.status = completed` (no ADR). Continue to §9.
  - **edit**: User wants to adjust content. Accept inline edits and re-present the draft. Re-open gate 2.

No gate after Sniper — content approval happens BEFORE the commit, not after.
```

Current language instruction:
```
**Instrução de idioma para Arquivista:** gerar o ADR no idioma definido em `active.language`.
- `language: pt` → conteúdo em português
- `language: en` → conteúdo em inglês
```
Replace with:
```
**Language instruction for Archivist:** generate the ADR in the language defined by `active.language.docs`.
- `docs: pt-BR` → content in Portuguese
- `docs: en` → content in English
```

Current ADR template section:
```
**Estrutura mínima do ADR (template para Arquivista):**
...
O template acima é em PT por padrão. Se `language: en`, Arquivista usa `Context`, `Decision`, `Consequences`.
```
Replace header and footer:
```
**Minimum ADR structure (template for Archivist):**
...
The template above is in PT-BR by default. If `docs: en`, Archivist uses `Context`, `Decision`, `Consequences`.
```

Translate ADR template fields. Current (PT):
```markdown
## Contexto
## Decisão
## Consequências
```
Add note that these vary by `active.language.docs`. Keep template as PT-BR default — it's a template for the agent, not user-facing prose.

**Section 8 Archivist display block:**
```
  📚 **Arquivista — rascunho de ADR:**
```
Replace with:
```
  📚 **Archivist — ADR draft:**
```

**Persona loading update** — add after the bootstrap fast-path config extraction block (§1), update instruction about `active.language`:

Find the line:
```
  - `active.language` → artifact language (`pt` if absent)
```
Replace with:
```
  - `active.language.chat` → agent chat response language (selects `content_by_lang` block in persona; fallback `pt-BR`)
  - `active.language.docs` → generated artifact language (passed to slot providers)
```

### Step 2: Sync the other two SKILL.md files

After completing the translation of `internal/embed/defaults/SKILL.md`, copy it to the other two locations:

```bash
cp internal/embed/defaults/SKILL.md .strategist/SKILL.md
cp internal/embed/defaults/SKILL.md strategist/SKILL.md
```

### Step 3: Run embed and strategist tests

```bash
go test ./internal/embed/... ./strategist/...
```

Expected: PASS. (These tests validate that embedded files are present and parseable, not prose content.)

### Step 4: Spot-check reserved words are intact

```bash
grep -n "saque rapido\|sim/nao\|analise\|entrega_revisada\|entrega_executada" internal/embed/defaults/SKILL.md | head -20
```

Expected: all five reserved word patterns appear in the output.

### Step 5: Commit

```bash
git add internal/embed/defaults/SKILL.md .strategist/SKILL.md strategist/SKILL.md
git commit -m "docs: rewrite SKILL.md to English-primary

Translates all instructional prose to English. No protocol or
behaviour changes. PT-BR reserved words preserved: saque rapido,
sim/nao, mission mode values (analise, entrega_revisada,
entrega_executada).

Updates persona loading instruction to use active.language.chat
as the content_by_lang selector."
```

---

## Final Verification

```bash
go test ./...
go build ./...
grep -rn "DoneScopeAnalise\|MissionModeAnalise\|MissionModeEntrega\|StateDoneAnalise\|StateDoneEntrega" internal/
```

Expected: zero matches from grep (all PT-BR constant names gone from Go code), all tests pass.

```bash
git log --oneline -4
```

Expected: four commits from this plan in order.
