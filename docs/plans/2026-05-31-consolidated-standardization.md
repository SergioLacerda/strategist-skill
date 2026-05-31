# Consolidated English Standardization + Embed Sync — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Standardize the codebase to English at every code/skill level while preserving PT-BR reserved words as an explicit bilingual protocol, and fix the embed sync gap that delivers a stale skill to users.

**Architecture:** Four independent streams. Stream 1 (Go domain constants) and Stream 2 (i18n package) have no dependencies. Stream 3 (personas `content_by_lang`) must complete before Stream 4 (SKILL.md rewrite), because SKILL.md references the persona template access pattern. All streams commit separately; merge is a no-conflict rewrite of separate files.

**Tech Stack:** Go 1.21+, `github.com/stretchr/testify/assert`, YAML, `go test ./...`, `go build ./...`

---

## Source of Truth Chain

```
strategist/           ← vendor canonical source (edit here first)
      ↓
internal/embed/defaults/   ← embedded default (copy from strategist after validation)
      ↓
.strategist/          ← user runtime deployment (copy from strategist after validation)
```

`internal/embed/defaults/` and `.strategist/` are outputs, never sources.

---

## Task 1: Rename PT-BR Go Domain Constants

**Files:**
- Modify: `internal/domain/policy.go`
- Modify: `internal/domain/state_machine.go`
- Modify: `internal/domain/state_machine_test.go`
- Modify: `internal/domain/policy_evaluator_test.go`

**Rename table:**

| Before | After | Value before | Value after | Note |
|---|---|---|---|---|
| `DoneScopeAnalise` | `DoneScopeAnalysis` | `"analise"` | `"analise"` | reserved word |
| `DoneScopeEntrega` | `DoneScopeDelivery` | `"entrega"` | `"entrega"` | reserved word |
| `MissionModeAnalise` | `MissionModeAnalysis` | `"analise"` | `"analise"` | reserved word |
| `MissionModeEntregaRevisada` | `MissionModeRevisedDelivery` | `"entrega_revisada"` | `"entrega_revisada"` | reserved word |
| `MissionModeEntregaExecutada` | `MissionModeExecutedDelivery` | `"entrega_executada"` | `"entrega_executada"` | reserved word |
| `StateDoneAnalise` | `StateDoneAnalysis` | `"DONE_ANALISE"` | `"DONE_ANALYSIS"` | internal FSM state |
| `StateDoneEntrega` | `StateDoneDelivery` | `"DONE_ENTREGA"` | `"DONE_DELIVERY"` | internal FSM state |

**Rule:** Reserved word values (`"analise"`, `"entrega"`, etc.) are part of the agent protocol — never change. Internal FSM state values (`"DONE_ANALISE"`) are code internals — always English.

### Step 1: Rewrite `internal/domain/policy.go`

Replace the entire file content:

```go
package domain

// Legacy done scopes.
const (
	DoneScopeAnalysis = "analise"
	DoneScopeDelivery = "entrega"
)

// Mission modes (single user-facing control).
const (
	MissionModeAnalysis         = "analise"
	MissionModeRevisedDelivery  = "entrega_revisada"
	MissionModeExecutedDelivery = "entrega_executada"
)

// Transition groups classify sensitive mission-state changes.
const (
	TransitionGroupFinalizeAnalysis = "finalize_analysis" // pending/refined -> done
	TransitionGroupExecution        = "execution"         // sniper/code/git/config writes
)

// MissionState represents the orchestrator FSM state.
type MissionState string

// Orchestrator finite-state machine states.
const (
	StateInit              MissionState = "INIT"
	StateOpportunityAttack MissionState = "OPPORTUNITY_ATTACK"
	StateOpportunityGate   MissionState = "OPPORTUNITY_GATE"
	StateOpportunityExec   MissionState = "OPPORTUNITY_EXEC"
	StateRefinement        MissionState = "REFINEMENT"
	StateApprovalGate      MissionState = "APPROVAL_GATE"
	StateExecution         MissionState = "EXECUTION"
	StateDoneAnalysis      MissionState = "DONE_ANALYSIS"
	StateDoneDelivery      MissionState = "DONE_DELIVERY"
	StateBlocked           MissionState = "BLOCKED"
)

// TransitionEvent represents FSM/evaluator inputs.
type TransitionEvent string

// Transition events accepted by the orchestrator state machine.
const (
	EventManifestEmpty    TransitionEvent = "manifest_empty"
	EventManifestNonEmpty TransitionEvent = "manifest_non_empty"
	EventGateApproved     TransitionEvent = "gate_approved"
	EventGateDenied       TransitionEvent = "gate_denied"
	EventSniperDone       TransitionEvent = "sniper_done"
	EventArchivistNoTasks TransitionEvent = "archivist_done_no_tasks"
	EventArchivistTasks   TransitionEvent = "archivist_done_has_tasks"
)

// MissionPolicy controls whether guarded transitions are allowed.
// MissionMode is canonical. DoneScope/ApplyChanges are derived for compatibility.
type MissionPolicy struct {
	Mode            string // analise | entrega_revisada | entrega_executada
	CanExecute      bool
	ExpectsDelivery string // analise | entrega
	DoneScope       string
	ApplyChanges    bool
}

// TransitionDecision is the deterministic result of policy evaluation.
type TransitionDecision struct {
	Allowed bool
	Reason  string
	Status  string // allowed | policy_blocked | approval_required
	Policy  MissionPolicy
}

// MissionModeFromLegacy maps the former 2-knob model to mission_mode.
func MissionModeFromLegacy(doneScope string, applyChanges bool) string {
	if doneScope == DoneScopeAnalysis {
		return MissionModeAnalysis
	}
	if applyChanges {
		return MissionModeExecutedDelivery
	}
	return MissionModeRevisedDelivery
}

// NewMissionPolicy builds canonical policy from mission_mode.
func NewMissionPolicy(mode string) MissionPolicy {
	switch mode {
	case MissionModeAnalysis:
		return MissionPolicy{Mode: mode, CanExecute: false, ExpectsDelivery: DoneScopeAnalysis, DoneScope: DoneScopeAnalysis, ApplyChanges: false}
	case MissionModeRevisedDelivery:
		return MissionPolicy{Mode: mode, CanExecute: false, ExpectsDelivery: DoneScopeDelivery, DoneScope: DoneScopeDelivery, ApplyChanges: false}
	case MissionModeExecutedDelivery:
		return MissionPolicy{Mode: mode, CanExecute: true, ExpectsDelivery: DoneScopeDelivery, DoneScope: DoneScopeDelivery, ApplyChanges: true}
	default:
		// Backward compatibility default preserves historical behavior.
		return NewMissionPolicy(MissionModeExecutedDelivery)
	}
}

// NormalizePolicy applies backward-compatible defaults/coherence.
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

### Step 2: Rewrite `internal/domain/state_machine.go`

```go
package domain

// NextState applies a single transition event to the current state using policy guards.
func NextState(current MissionState, event TransitionEvent, policy MissionPolicy) MissionState {
	p := NormalizePolicy(policy)

	switch current {
	case StateInit:
		return nextFromInit(event)
	case StateOpportunityAttack:
		return nextFromOpportunityAttack(event)
	case StateOpportunityGate:
		return nextFromOpportunityGate(event, p)
	case StateOpportunityExec:
		return nextFromOpportunityExec(event)
	case StateRefinement:
		return nextFromRefinement(event, p)
	case StateApprovalGate:
		return nextFromApprovalGate(event, p)
	case StateExecution:
		return nextFromExecution(event)
	case StateDoneAnalysis:
		return StateDoneAnalysis
	case StateDoneDelivery:
		return StateDoneDelivery
	case StateBlocked:
		return StateBlocked
	}

	return current
}

func nextFromInit(event TransitionEvent) MissionState {
	switch event {
	case EventManifestEmpty, EventManifestNonEmpty:
		return StateOpportunityAttack
	case EventGateApproved, EventGateDenied, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks:
		return StateInit
	}
	return StateInit
}

func nextFromOpportunityAttack(event TransitionEvent) MissionState {
	switch event {
	case EventManifestEmpty:
		return StateRefinement
	case EventManifestNonEmpty:
		return StateOpportunityGate
	case EventGateApproved, EventGateDenied, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks:
		return StateOpportunityAttack
	}
	return StateOpportunityAttack
}

func nextFromOpportunityGate(event TransitionEvent, p MissionPolicy) MissionState {
	switch event {
	case EventGateDenied:
		return StateRefinement
	case EventGateApproved:
		if p.CanExecute {
			return StateOpportunityExec
		}
		return StateRefinement
	case EventManifestEmpty, EventManifestNonEmpty, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks:
		return StateOpportunityGate
	}
	return StateOpportunityGate
}

func nextFromOpportunityExec(event TransitionEvent) MissionState {
	switch event {
	case EventSniperDone:
		return StateRefinement
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventArchivistNoTasks, EventArchivistTasks:
		return StateOpportunityExec
	}
	return StateOpportunityExec
}

func nextFromRefinement(event TransitionEvent, p MissionPolicy) MissionState {
	switch event {
	case EventArchivistNoTasks:
		return StateDoneAnalysis
	case EventArchivistTasks:
		if p.Mode == MissionModeAnalysis {
			return StateDoneAnalysis
		}
		return StateApprovalGate
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventSniperDone:
		return StateRefinement
	}
	return StateRefinement
}

func nextFromApprovalGate(event TransitionEvent, p MissionPolicy) MissionState {
	switch event {
	case EventGateDenied:
		return StateDoneAnalysis
	case EventGateApproved:
		if p.CanExecute {
			return StateExecution
		}
		return StateDoneDelivery
	case EventManifestEmpty, EventManifestNonEmpty, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks:
		return StateApprovalGate
	}
	return StateApprovalGate
}

func nextFromExecution(event TransitionEvent) MissionState {
	switch event {
	case EventSniperDone:
		return StateDoneDelivery
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventArchivistNoTasks, EventArchivistTasks:
		return StateExecution
	}
	return StateExecution
}

// RunStateMachine folds a sequence of events from a starting state.
func RunStateMachine(start MissionState, events []TransitionEvent, policy MissionPolicy) MissionState {
	state := start
	for _, ev := range events {
		state = NextState(state, ev, policy)
	}
	return state
}
```

### Step 3: Rewrite `internal/domain/state_machine_test.go`

```go
package domain_test

import (
	"math/rand"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestFSMAnaliseNeverExecutes(t *testing.T) {
	t.Parallel()
	policy := domain.NewMissionPolicy(domain.MissionModeAnalysis)
	state := domain.StateInit
	events := []domain.TransitionEvent{
		domain.EventManifestNonEmpty,
		domain.EventGateApproved,
		domain.EventArchivistTasks,
		domain.EventGateApproved,
		domain.EventSniperDone,
	}
	for _, ev := range events {
		state = domain.NextState(state, ev, policy)
		assert.NotEqual(t, domain.StateExecution, state)
	}
}

func TestOpportunityGatePolicyLocked(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{domain.MissionModeAnalysis, domain.MissionModeRevisedDelivery} {
		state := domain.RunStateMachine(domain.StateOpportunityAttack,
			[]domain.TransitionEvent{domain.EventManifestNonEmpty, domain.EventGateApproved},
			domain.NewMissionPolicy(mode),
		)
		assert.Equal(t, domain.StateRefinement, state)
	}
}

func TestFSMSafetyPropertyLike(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(31))
	allEvents := []domain.TransitionEvent{
		domain.EventManifestEmpty,
		domain.EventManifestNonEmpty,
		domain.EventGateApproved,
		domain.EventGateDenied,
		domain.EventSniperDone,
		domain.EventArchivistNoTasks,
		domain.EventArchivistTasks,
	}

	for i := 0; i < 400; i++ {
		mode := domain.MissionModeExecutedDelivery
		if rng.Intn(3) == 0 {
			mode = domain.MissionModeAnalysis
		} else if rng.Intn(2) == 0 {
			mode = domain.MissionModeRevisedDelivery
		}
		policy := domain.NewMissionPolicy(mode)
		state := domain.StateInit
		seenGateApproved := false
		for j := 0; j < 14; j++ {
			ev := allEvents[rng.Intn(len(allEvents))]
			if ev == domain.EventGateApproved {
				seenGateApproved = true
			}
			state = domain.NextState(state, ev, policy)
			if state == domain.StateExecution {
				assert.True(t, seenGateApproved)
				assert.True(t, policy.CanExecute)
			}
		}
	}
}
```

### Step 4: Rewrite `internal/domain/policy_evaluator_test.go`

```go
package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestGuardedTransitionRequiresGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeDelivery, ApplyChanges: true},
		domain.TransitionGroupExecution,
		false,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "approval_required", decision.Reason)
}

func TestDefaultMissionModePreservesLegacyExecution(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{},
		domain.TransitionGroupExecution,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
	assert.Equal(t, domain.MissionModeExecutedDelivery, decision.Policy.Mode)
}

func TestDoneAnaliseSkipsExecution(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeAnalysis, ApplyChanges: true},
		domain.TransitionGroupExecution,
		true,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "policy_blocked", decision.Reason)
}

func TestExecutionAllowedWhenEntregaAndApplyChanges(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeDelivery, ApplyChanges: true},
		domain.TransitionGroupExecution,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}

func TestFinalizeAnalysisAllowedWithGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeAnalysis, ApplyChanges: false},
		domain.TransitionGroupFinalizeAnalysis,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}

func TestIncidentUXStratetist(t *testing.T) {
	t.Parallel()

	// Regression: no explicit approval must never allow execution-like transition.
	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeDelivery, ApplyChanges: true},
		domain.TransitionGroupExecution,
		false,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "approval_required", decision.Reason)
}

func TestQuickDrawPolicyLockedForAnalysisMode(t *testing.T) {
	t.Parallel()

	// Quick draw execution is an execution-like guarded transition.
	decision := domain.EvaluateGuardedTransition(
		domain.NewMissionPolicy(domain.MissionModeAnalysis),
		domain.TransitionGroupExecution,
		true,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "policy_blocked", decision.Reason)
}

func TestQuickDrawAllowedForEntregaExecutadaWithGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.NewMissionPolicy(domain.MissionModeExecutedDelivery),
		domain.TransitionGroupExecution,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}
```

### Step 5: Build and test

```bash
go build ./...
go test ./internal/domain/...
```

Expected: all tests pass, zero compile errors.

### Step 6: Commit Task 1

```bash
git add internal/domain/policy.go internal/domain/state_machine.go \
        internal/domain/state_machine_test.go internal/domain/policy_evaluator_test.go
git commit -m "refactor: rename PT-BR Go domain identifiers to English

Constants renamed; reserved word string values preserved.
StateDone* values updated from DONE_ANALISE/DONE_ENTREGA to DONE_ANALYSIS/DONE_DELIVERY
(internal FSM states, never persisted to YAML)."
```

---

## Task 2: Create `internal/i18n/` Package

**Files:**
- Create: `internal/i18n/wizard_test.go`
- Create: `internal/i18n/wizard.go`
- Create: `internal/i18n/reserved.go`
- Modify: `internal/install/wizard.go`

### Step 1: Write the failing test first

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

func TestBundleForCaseInsensitive(t *testing.T) {
	assert.Equal(t, i18n.BundleFor("pt-BR"), i18n.BundleFor("PT-BR"))
	assert.Equal(t, i18n.BundleFor("pt-BR"), i18n.BundleFor("pt-br"))
}

func TestAllFieldsNonEmptyEN(t *testing.T) {
	b := i18n.EN
	v := reflect.ValueOf(b)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i).Name
		assert.NotEmpty(t, v.Field(i).String(), "EN.%s must not be empty", field)
	}
}

func TestAllFieldsNonEmptyPT(t *testing.T) {
	b := i18n.PT
	v := reflect.ValueOf(b)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i).Name
		assert.NotEmpty(t, v.Field(i).String(), "PT.%s must not be empty", field)
	}
}
```

### Step 2: Run test — expect compile failure

```bash
go test ./internal/i18n/...
```

Expected: `cannot find package "github.com/SergioLacerda/strategist-skill/internal/i18n"`

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

// EN is the English wizard string bundle.
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

// PT is the Portuguese wizard string bundle.
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

### Step 4: Create `internal/i18n/reserved.go`

```go
package i18n

// PT-BR reserved words used as agent match tokens and YAML config values.
// These are NOT user-visible strings — they are part of the bilingual protocol
// between the user, active.yaml, and the agent. Do not translate.
const (
	// MissionMode values — written to active.yaml, matched by the agent.
	ReservedMissionModeAnalysis        = "analise"
	ReservedMissionModeRevisedDelivery = "entrega_revisada"
	ReservedMissionModeExecDelivery    = "entrega_executada"
	ReservedDoneScopeDelivery          = "entrega"

	// Gate responses — typed by the user, matched by the agent.
	ReservedGateYes = "sim"
	ReservedGateNo  = "nao"

	// Quick draw triggers — typed by the user, matched by the agent.
	ReservedQuickDrawPT = "saque rapido"
	ReservedQuickDrawEN = "quick draw"
)
```

### Step 5: Run tests — expect pass

```bash
go test ./internal/i18n/...
```

Expected: all 6 tests pass.

### Step 6: Update `internal/install/wizard.go`

Changes:
- Remove lines 10–63: `wizardStrings` struct, `bundleEN`, `bundlePTBR`, `bundleFor()`
- Add import: `"github.com/SergioLacerda/strategist-skill/internal/i18n"`
- Change `b := bundleFor(uiLang)` → `b := i18n.BundleFor(uiLang)`
- Remove `customLabel := "(digitar outro...")`
- In all three `SelectOrInput` calls, replace `customLabel` → `b.LabelCustomInput`

Complete `internal/install/wizard.go` after changes:

```go
package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

var langOptions = []string{"en", "pt-BR"}

// runWizard collects install configuration through p.
func runWizard(p Prompter) (domain.WizardConfig, error) {
	// Prompt 1 — bilingual, bundle not yet chosen
	uiLang, err := p.Select("Preferred language / Idioma preferido", "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: ui_language: %w", err)
	}
	uiLang = normLang(uiLang)

	b := i18n.BundleFor(uiLang)

	docLang, err := p.Select(b.PromptDocLang, "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: doc_language: %w", err)
	}
	docLang = normLang(docLang)

	chatLang, err := p.Select(b.PromptChatLang, "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: chat_language: %w", err)
	}
	chatLang = normLang(chatLang)

	codeLang, err := p.Select(b.PromptCodeLang, "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: code_language: %w", err)
	}
	codeLang = normLang(codeLang)

	mode, err := p.Select(b.PromptMode, "pragmatic", []string{"pragmatic", "epic"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: mode: %w", err)
	}

	basePath, err := p.Input(b.PromptBasePath, ".analysis")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: base_path: %w", err)
	}

	adrRaw, err := p.Select(b.PromptAdr, "yes", []string{"yes", "no"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: adr_enabled: %w", err)
	}
	adrEnabled := adrRaw == "yes"

	missionMode, err := p.Select(b.PromptMissionMode, "entrega_executada", []string{"analise", "entrega_revisada", "entrega_executada"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: mission_mode: %w", err)
	}

	policy := domain.NewMissionPolicy(missionMode)
	doneScope := policy.DoneScope
	applyChanges := policy.ApplyChanges

	fmt.Println(b.HeaderSlots)

	discovery, err := p.SelectOrInput(b.PromptDiscovery, "brainstorming", []string{"brainstorming"}, b.LabelCustomInput)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: discovery: %w", err)
	}

	refinement, err := p.SelectOrInput(b.PromptRefinement, "openspec-explore", []string{"openspec-explore"}, b.LabelCustomInput)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: refinement: %w", err)
	}

	execution, err := p.SelectOrInput(b.PromptExecution, "sdd-ask", []string{"sdd-ask", "sdd-ask-full"}, b.LabelCustomInput)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: execution: %w", err)
	}

	fmt.Println(b.HeaderChest)

	chestPath, err := p.Input(b.PromptChestPath, "")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: treasure_chest: %w", err)
	}

	return domain.WizardConfig{
		Mode:               mode,
		BasePath:           basePath,
		MissionMode:        missionMode,
		DoneScope:          doneScope,
		ApplyChanges:       applyChanges,
		UILanguage:         uiLang,
		DocLanguage:        docLang,
		ChatLanguage:       chatLang,
		CodeLanguage:       codeLang,
		AdrEnabled:         adrEnabled,
		DiscoveryProvider:  discovery,
		RefinementProvider: refinement,
		ExecutionProvider:  execution,
		TreasureChestPath:  chestPath,
	}, nil
}

// normLang normalises language input to canonical form: "en" or "pt-BR".
func normLang(raw string) string {
	if strings.EqualFold(raw, "pt-BR") {
		return "pt-BR"
	}
	return raw
}
```

### Step 7: Verify

```bash
go build ./...
go test ./internal/i18n/... ./internal/install/...

# No hardcoded PT-BR label in wizard
grep -n "digitar outro" internal/install/wizard.go
```

Expected: tests pass, grep returns no output.

### Step 8: Commit Task 2

```bash
git add internal/i18n/ internal/install/wizard.go
git commit -m "feat: extract internal/i18n package with WizardStrings and PT-BR reserved words

Moves wizard i18n bundle from unexported install package to exported i18n package.
Adds LabelCustomInput field (was hardcoded PT-BR in wizard.go).
Adds reserved.go documenting PT-BR protocol tokens that must never be translated."
```

---

## Task 3: Persona `content_by_lang` Restructure

**Files:**
- Rewrite: `internal/embed/defaults/personas/epic.yaml`
- Rewrite: `internal/embed/defaults/personas/pragmatic.yaml`
- Sync: copy both to `strategist/personas/`

**Note on `quick_draw_gate`:** `(sim/nao)` is kept in BOTH language blocks — it is a runtime reserved word the user types, not a UI string.

### Step 1: Rewrite `internal/embed/defaults/personas/epic.yaml`

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
  Senior architect who has lived long enough to know the work is hard — and chose
  to find joy in it anyway. Believes the era of human and digital agents collaborating
  is one of the greatest adventures engineering has ever seen. Narrates phases with
  genuine energy: Ranger, Archivist, Sniper — each role has its moment. Keeps
  technical vocabulary (commit, analysis, implementation, mission) — the epic is in
  the roles and the narrative, not in replacing vocabulary. Keeps the team together.
  When something fails, it is a tough boss, not a tragedy.

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
      🎯 **Ranger:** reconnaissance complete. Artifact at: {artifact_path}

    archivist_start: >
      📚 **Archivist:** starting analysis and refinement. skill={provider}
    archivist_done: >
      📚 **Archivist:** analysis refined. Artifacts at: {artifact_path}

    sniper_start: >
      🗡️ **Sniper:** mission approved — starting implementation.
    sniper_done: >
      🗡️ **Sniper:** implementation complete. Report at: {artifact_path}

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

### Step 2: Rewrite `internal/embed/defaults/personas/pragmatic.yaml`

```yaml
id: pragmatic
description: >
  Senior architect — direct, no fluff. For those who prefer facts over narrative.

phase_labels:
  discovery: analyst
  refinement: analyst
  execution: developer

tone_directive: >
  Senior architect, fifteen years of scars. Narrates phases because someone has to
  orchestrate — not because they enjoy it. No gamification, no euphoria, no flattering
  adjectives. Speaks of roles as analyst (discovery and analysis) and developer
  (execution). "Sprint" instead of "mission". Short language: subject + verb + artifact.
  Problems are problems — no drama, no softening. One line per event where possible.

progress_prefix: "[Strategist]"

content_by_lang:
  en:
    intake_summary: >
      Sprint: {task_type} | delivery={delivery_strategy} | compat={legacy_compatibility} |
      urgency={urgency} | intent={execution_intent}

    ranger_start: >
      Analyst started discovery. Provider: {provider}.
    ranger_done: >
      Discovery done. Artifact: {artifact_path}

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
      Pending:
      {manifest}
      Approve? (yes / no / select)

    quick_draw_detected: >
      Quick draw detected. Short side quest started (Analyst -> Analyst -> Gate).
    quick_draw_gate: >
      idea: {idea}

      add idea? (sim/nao)
    quick_draw_success: >
      success: idea added at {destination_path}
      total ideas: {total_ideas}
      similar ideas (same theme): {similar_ideas}

    treasure_chest_found: >
      Chest: [{chest_id}] — {description}
    side_quest_detected: >
      Side quest: {description}
    opportunity_signal: >
      {count} opportunity(ies) detected.

    mission_checkpoint: |
      Pipeline — {mission_id}
      {step_1_icon} 1 — analyst
      {step_2_icon} 2 — analyst
      {step_3_icon} 3 — gate
      {step_4_icon} 4 — developer

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

### Step 3: Sync to `strategist/personas/`

```bash
cp internal/embed/defaults/personas/epic.yaml strategist/personas/epic.yaml
cp internal/embed/defaults/personas/pragmatic.yaml strategist/personas/pragmatic.yaml
```

### Step 4: Verify byte-identical

```bash
diff internal/embed/defaults/personas/epic.yaml strategist/personas/epic.yaml
diff internal/embed/defaults/personas/pragmatic.yaml strategist/personas/pragmatic.yaml
```

Expected: no output.

### Step 5: Build check

```bash
go build ./...
go test ./internal/embed/...
```

Expected: pass.

### Step 6: Commit Task 3

```bash
git add internal/embed/defaults/personas/ strategist/personas/
git commit -m "feat: restructure personas to content_by_lang with EN and PT-BR blocks

Adds English chat template set to both epic and pragmatic personas.
Translates description and tone_directive to English (agent instructions).
PT-BR templates preserved verbatim under content_by_lang.pt-BR.
Syncs strategist/personas/ to embed/defaults/personas/ (byte-identical)."
```

---

## Task 4: SKILL.md Bidirectional Merge + Full English Rewrite

**Canonical file:** `strategist/SKILL.md` (edit here; propagate to embed/ and .strategist/ at end)

**Sub-task A** merges the three blocks that exist only in `internal/embed/defaults/SKILL.md` into `strategist/SKILL.md`.  
**Sub-task B** does the full English rewrite and `content_by_lang` migration on the merged file.

### Sub-task A: Merge embed-exclusive content into `strategist/SKILL.md`

#### Step A1: Find §5.0 insertion point

```bash
grep -n "^### 5\.0 Quick Draw\|^## 5\. Mission" strategist/SKILL.md | head -5
```

Note the line number where `### 5.0 Quick Draw Side Quest (conditional)` begins.

#### Step A2: Insert §5.-1 before §5.0

Immediately before the `### 5.0 Quick Draw Side Quest` line, insert:

```markdown
### 5.-1 Mandatory Opportunity Sweep Invariant

In every mission phase, Strategist MUST perform and report:
- `opportunity_scan=done`
- `treasure_check=done`
- `sidequest_manifest=updated|empty`

This invariant applies even for narrow prompts (single-file/single-target refinement).
"Foco em alvo único" is NOT a valid reason to skip sweeps.

If a sweep cannot run due to technical error, emit:
`[Strategist] phase=<phase_label> status=blocked reason=opportunity_sweep_failed`
and stop.

```

#### Step A3: Find §5.0b Archivist append location

```bash
grep -n "scope_addition\|Sniper.*append\|quick_draw.*append\|5\.0d\|step.*append" strategist/SKILL.md | head -10
```

Find the line in §5.0b where the Archivist appends the idea (`scope_addition` operation). Insert this block immediately before the append instruction:

```markdown
Before append, evaluate guarded transition group `finalize_analysis` with effective policy.
Emit canonical event:
`[Strategist] phase=policy_eval status=<allowed|blocked> mission=<id> mode=<mode> can_execute=<bool> transition_group=finalize_analysis`.
If blocked, stop with `reason=policy_blocked`.

```

#### Step A4: Find §5d opportunity execution location

```bash
grep -n "^### 5d\|^#### 5d\|Gate de Oportunidade\|Execução de Oportun\|opportunity.*execu" strategist/SKILL.md | head -5
```

Find the §5d heading (opportunity execution). Insert this block immediately before it:

```markdown
Before 5d, evaluate guarded transition group `execution` with effective policy.
Emit canonical event:
`[Strategist] phase=policy_eval status=<allowed|blocked> mission=<id> mode=<mode> can_execute=<bool> transition_group=execution`.
If blocked, skip opportunity execution and continue to 5e with `execution_skipped_by_policy`.

```

#### Step A5: Verify merge

```bash
grep -c "opportunity_sweep_failed\|transition_group=finalize_analysis\|transition_group=execution" strategist/SKILL.md
```

Expected: `3`

#### Step A6: Commit merge

```bash
git add strategist/SKILL.md
git commit -m "chore: merge embed-exclusive SKILL.md content into strategist/ canonical source

Adds §5.-1 Mandatory Opportunity Sweep Invariant and policy_eval emit events
for finalize_analysis and execution transition groups."
```

### Sub-task B: Full English Rewrite

#### Step B1: Fix `active.language` scalar references

```bash
grep -n "active\.language[^.]" strategist/SKILL.md
```

Two occurrences in the bootstrap section. Replace:

```
active.language → artifact language (pt if absent)
```
→
```
active.language.chat → chat language for persona template selection (default: pt-BR)
```

```
Extract active.language (default: pt) — pass to all slot providers and use for artifact generation.
```
→
```
Extract active.language (object with keys: ui, docs, chat, code).
Pass active.language.docs to slot providers for artifact generation.
Use active.language.chat for persona template selection (default: pt-BR if absent).
```

#### Step B2: Add persona access note and replace all `prompt_templates`

After the `## 3.` header (or §3.1 start), add one note block:

```
> **Persona template access:** `persona.content_by_lang[active.language.chat].<key>`.
> Fallback: if `active.language.chat` is absent or has no matching block, use `pt-BR`.
```

Count occurrences before:

```bash
grep -c "prompt_templates" strategist/SKILL.md
```

Replace every occurrence of `persona.prompt_templates.` with `persona.content_by_lang[active.language.chat].`

Verify:

```bash
grep -c "prompt_templates" strategist/SKILL.md
```

Expected: `0`

#### Step B3: Bucket i18n in §5.0b

```bash
grep -n "arquitetura\|seguranca\|analise.*geral\|todo/<tema" strategist/SKILL.md
```

Replace the hardcoded PT-BR bucket block:

```
- Determine theme from lightweight buckets:
  - `arquitetura`, `seguranca`, `analise`, `geral`
- Resolve destination path:
  - `<base_path>/todo/<tema>.md` (e.g. `.analysis/todo/arquitetura.md`)
```

With:

```
- Determine theme bucket based on `active.language.chat`:
  - pt-BR: `arquitetura` | `seguranca` | `analise` | `geral`
  - en:    `architecture` | `security` | `analysis` | `general`
- Resolve destination path: `<base_path>/todo/<bucket>.md`
  - pt-BR example: `.analysis/todo/arquitetura.md`
  - en example: `.analysis/todo/architecture.md`
```

Apply the same i18n note to any other `todo/<slug>.md` reference with PT-BR bucket names.

#### Step B4: Translate PT-BR prose

Find and replace each entry. Search with `grep -n` first to confirm location, then edit:

| Find | Replace |
|---|---|
| `substitui {provider} com o skill id do provider` | `substitute {provider} with the slot provider skill id` |
| `substitui {artifact_path}` | `substitute {artifact_path}` |
| `substitui {mission_id}` | `substitute {mission_id}` |
| `com {artifact_path}` | `with {artifact_path}` |
| `Sinalização de baú:` | `Chest signal:` |
| `Instrução de idioma para Arquivista:` | `Language instruction for Archivist:` |
| `gerar o ADR no idioma definido em` | `generate the ADR in the language defined by` |
| `Critérios de ativação — avaliar se a missão contém decisões arquiteturais` | `Activation criteria — evaluate if the mission contains architectural decisions` |
| `rascunho de ADR` | `ADR draft` |
| `Commitar? (yes / no)` | `Commit? (yes / no)` |
| `Checklist de Missão e Fases por Papel` | `Mission Checklist and Phase Roles` |
| `concluído` | `done` |
| `pendente` | `pending` |
| `não aplicável` | `not applicable` |
| `pós-missão, condicional` | `post-mission, conditional` |
| `(missão futura)` | `(future mission)` |
| `Criar <base_path>/todo/<slug>.md com o escopo adicional detectado` | `Create <base_path>/todo/<slug>.md with the detected additional scope` |
| `Missão encerrada` | `Mission closed` |
| `Aguardar resposta` | `Wait for response` |

#### Step B5: Update §3.2 Mission Checkpoint section header and table

Ensure the section reads:

```markdown
### 3.2 Mission Checkpoint

After intake completes, initialize and emit the mission pipeline checkpoint using
`persona.content_by_lang[active.language.chat].mission_checkpoint` with:
- `{mission_id}` = the generated mission id
- `{step_1_icon}` = `⏳` (Ranger about to start), `{step_2_icon}` = `{step_3_icon}` = `{step_4_icon}` = `⬜`

Re-emit the checkpoint at each phase transition, updating icons:

| After phase    | step_1 | step_2 | step_3 | step_4 |
|----------------|--------|--------|--------|--------|
| Intake         | ⏳     | ⬜     | ⬜     | ⬜     |
| Ranger done    | ✅     | ⏳     | ⬜     | ⬜     |
| Archivist done | ✅     | ✅     | ⏳     | ⬜     |
| Gate approved  | ✅     | ✅     | ✅     | ⏳     |
| Sniper done    | ✅     | ✅     | ✅     | ✅     |

Icons: `⏳` = running, `✅` = done, `⬜` = pending.
Skip re-emit when mission ends at `plan_only` (gate declined).
```

#### Step B6: Verify clean

```bash
# No prompt_templates
grep -c "prompt_templates" strategist/SKILL.md

# No scalar active.language
grep -n "active\.language[^.]" strategist/SKILL.md

# No PT-BR prose (reserved words allowed: sim/nao, saque rapido, analise, entrega*)
grep -in "Missão\|missão\|Artefato em\|Aguardar\|Sinalização\|concluído\|não aplicável\|Commitar\|pós-missão\|missão futura\|Instrução de idioma\|Critérios de ativação\|rascunho de ADR" strategist/SKILL.md
```

Expected for each: `0` / no output.

#### Step B7: Propagate

```bash
cp strategist/SKILL.md internal/embed/defaults/SKILL.md
cp strategist/SKILL.md .strategist/SKILL.md
```

#### Step B8: Verify byte-identical

```bash
md5sum strategist/SKILL.md internal/embed/defaults/SKILL.md .strategist/SKILL.md
```

Expected: all three hashes match.

#### Step B9: Final build and test

```bash
go build ./...
go test ./...
```

Expected: all tests pass.

#### Step B10: Commit

```bash
git add strategist/SKILL.md internal/embed/defaults/SKILL.md .strategist/SKILL.md
git commit -m "feat: rewrite SKILL.md to English with content_by_lang and bucket i18n

All instructional prose translated to English.
active.language scalar references replaced with object form (.chat/.docs).
prompt_templates replaced with content_by_lang[active.language.chat].
Bucket names now language-aware via active.language.chat.
Three SKILL.md copies synced to byte-identical state."
```

---

## Final Verification Checklist

```bash
# 1. Build clean
go build ./...

# 2. All tests pass
go test ./...

# 3. No PT-BR identifiers in Go domain code
grep -rn "Analise\|Entrega\b" --include="*.go" internal/domain/

# 4. No hardcoded PT-BR in wizard
grep -n "digitar outro" internal/install/wizard.go

# 5. No prompt_templates in any SKILL.md
grep -c "prompt_templates" strategist/SKILL.md internal/embed/defaults/SKILL.md .strategist/SKILL.md

# 6. No scalar active.language in SKILL.md
grep -n "active\.language[^.]" strategist/SKILL.md

# 7. Three SKILL.md copies byte-identical
md5sum strategist/SKILL.md internal/embed/defaults/SKILL.md .strategist/SKILL.md

# 8. Both personas byte-identical between strategist/ and embed/
diff strategist/personas/epic.yaml internal/embed/defaults/personas/epic.yaml
diff strategist/personas/pragmatic.yaml internal/embed/defaults/personas/pragmatic.yaml

# 9. i18n reserved.go in place
grep -c "ReservedMissionModeAnalysis\|ReservedGateYes\|ReservedQuickDrawPT" internal/i18n/reserved.go
```

All expected outputs: `0` / no diff / count of `3` for step 9.
