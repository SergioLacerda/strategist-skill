package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func coverageCompiledRegistry() CompiledRegistry {
	return CompiledRegistry{
		SchemaVersion: CompiledRegistrySchemaVersion,
		Weapons: []CompiledWeapon{{
			ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon", SourceDigest: "sha256:source",
			Origin:  WeaponOriginEmbedded,
			Runtime: WeaponRuntime{Kind: RankedRuntimeEmbedded, ExecutionMode: WeaponExecutionModePromptBridge},
		}},
		Roles: []CompiledRole{{ID: "ranger", Slot: "discovery", ContractDigest: "sha256:role"}},
		RankedBindings: []CompiledRankedBinding{{
			Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0",
			WeaponDigest: "sha256:weapon", RoleDigest: "sha256:role", BindingDigest: "sha256:binding",
			CertificationDigest: "sha256:cert", SourceDigest: "sha256:source",
			ExecutionMode: WeaponExecutionModePromptBridge, ConnectorID: "embedded", Entrypoint: "discover",
			Runtime:    WeaponRuntime{Kind: RankedRuntimeEmbedded, ExecutionMode: WeaponExecutionModePromptBridge},
			Generation: 1, Status: "active",
		}},
	}
}

func TestCompiledRegistryDocumentAndCompatibilityCoverage(t *testing.T) {
	document := CompiledRegistryDocument{
		SchemaVersion: "catalog/v1", Weapons: coverageCompiledRegistry().Weapons, Roles: coverageCompiledRegistry().Roles,
		Compatibility: []CompiledCompatibility{{
			Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0",
			WeaponDigest: "sha256:weapon", Source: "embedded",
		}}, RankedBindings: coverageCompiledRegistry().RankedBindings,
	}
	registry, err := CompiledRegistryFromDocument(document)
	require.NoError(t, err)
	offers := registry.RankedBindingsFor("ranger", "discovery")
	require.Len(t, offers, 1)
	require.Empty(t, registry.RankedBindingsFor("missing", "discovery"))
	assert.Equal(t, CanonicalTaxonomyVersion, registry.TaxonomyVersion)
	assert.Equal(t, "brainstorming@1.0.0", WeaponRef("brainstorming", "1.0.0"))
	assert.Equal(t, "brainstorming", WeaponRef("brainstorming", ""))
	assert.Equal(t, "brainstorming@0.0.0", WeaponPayloadDirName("brainstorming", ""))
	assert.Equal(t, "brainstorming@1.0.0", WeaponPayloadDirName("brainstorming", "1.0.0"))

	_, err = CompiledRegistryFromDocument(CompiledRegistryDocument{})
	require.ErrorContains(t, err, "schema_version is required")
}

func TestCompiledRegistryValidationErrorCoverage(t *testing.T) {
	cases := map[string]func(*CompiledRegistry){
		"unsupported schema":  func(reg *CompiledRegistry) { reg.SchemaVersion = "wrong" },
		"empty weapons":       func(reg *CompiledRegistry) { reg.Weapons = nil },
		"empty roles":         func(reg *CompiledRegistry) { reg.Roles = nil },
		"missing weapon id":   func(reg *CompiledRegistry) { reg.Weapons[0].ID = "" },
		"missing digest":      func(reg *CompiledRegistry) { reg.Weapons[0].Digest = "" },
		"invalid origin":      func(reg *CompiledRegistry) { reg.Weapons[0].Origin = "foreign" },
		"invalid runtime":     func(reg *CompiledRegistry) { reg.Weapons[0].Runtime.Kind = "foreign" },
		"embedded mode":       func(reg *CompiledRegistry) { reg.Weapons[0].Runtime.ExecutionMode = "" },
		"missing source":      func(reg *CompiledRegistry) { reg.Weapons[0].SourceDigest = "" },
		"missing role id":     func(reg *CompiledRegistry) { reg.Roles[0].ID = "" },
		"invalid role slot":   func(reg *CompiledRegistry) { reg.Roles[0].Slot = "foreign" },
		"missing role digest": func(reg *CompiledRegistry) { reg.Roles[0].ContractDigest = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			registry := coverageCompiledRegistry()
			mutate(&registry)
			require.Error(t, registry.Validate())
		})
	}

	compatibilityCases := map[string]func(*CompiledRegistry){
		"unknown role": func(reg *CompiledRegistry) {
			reg.Compatibility = []CompiledCompatibility{{Role: "missing", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0", WeaponDigest: "sha256:weapon", Source: "embedded"}}
		},
		"unknown weapon": func(reg *CompiledRegistry) {
			reg.Compatibility = []CompiledCompatibility{{Role: "ranger", Slot: "discovery", WeaponID: "missing", WeaponVersion: "1.0.0", WeaponDigest: "sha256:weapon", Source: "embedded"}}
		},
		"digest mismatch": func(reg *CompiledRegistry) {
			reg.Compatibility = []CompiledCompatibility{{Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0", WeaponDigest: "sha256:other", Source: "embedded"}}
		},
		"missing source": func(reg *CompiledRegistry) {
			reg.Compatibility = []CompiledCompatibility{{Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0", WeaponDigest: "sha256:weapon"}}
		},
	}
	for name, mutate := range compatibilityCases {
		t.Run("compatibility "+name, func(t *testing.T) {
			registry := coverageCompiledRegistry()
			mutate(&registry)
			require.Error(t, registry.Validate())
		})
	}
}

func TestCompiledRegistryRankedBindingValidationErrorCoverage(t *testing.T) {
	cases := map[string]func(*CompiledRegistry){
		"unknown role":       func(reg *CompiledRegistry) { reg.RankedBindings[0].Role = "missing" },
		"missing version":    func(reg *CompiledRegistry) { reg.RankedBindings[0].WeaponVersion = "" },
		"incomplete digest":  func(reg *CompiledRegistry) { reg.RankedBindings[0].BindingDigest = "" },
		"execution identity": func(reg *CompiledRegistry) { reg.RankedBindings[0].ExecutionMode = WeaponExecutionModeCode },
		"runtime identity":   func(reg *CompiledRegistry) { reg.RankedBindings[0].ConnectorID = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			registry := coverageCompiledRegistry()
			mutate(&registry)
			require.Error(t, registry.Validate())
		})
	}
}

func TestMissionInvocationStateAndPrimitiveCoverage(t *testing.T) {
	assert.True(t, InvocationStatePending.CanTransitionTo(InvocationStateProcessing))
	assert.False(t, InvocationStatePending.CanTransitionTo(InvocationStateCompleted))
	assert.True(t, InvocationStateProcessing.CanTransitionTo(InvocationStateProcessing))
	assert.True(t, InvocationStateProcessing.CanTransitionTo(InvocationStateCompleted))
	assert.False(t, InvocationStateCompleted.CanTransitionTo(InvocationStateProcessing))
	assert.False(t, MissionInvocationState("future").CanTransitionTo(InvocationStateProcessing))

	assert.Equal(t, InvocationStatePending, (MissionInvocationRecord{}).EffectiveState())
	assert.Equal(t, InvocationStateCompleted, (MissionInvocationRecord{Consumed: true}).EffectiveState())
	assert.Equal(t, InvocationStateProcessing, (MissionInvocationRecord{State: InvocationStateProcessing, Consumed: true}).EffectiveState())
	assert.Equal(t, ExecutionAdapterCurrentHostUnverified, (MissionInvocationRecord{}).EffectiveAdapter())
	assert.Equal(t, ExecutionAdapterCodexChild, (MissionInvocationRecord{ExecutionAdapter: ExecutionAdapterCodexChild}).EffectiveAdapter())

	assert.Equal(t, []string{EvidenceClassExplicit, EvidenceClassCorroboratedInference, EvidenceClassWeakInference, EvidenceClassUnknown}, EvidenceClasses())
	assert.Equal(t, "brainstorming@1.0.0", (SlotBinding{Mode: SlotBindingModeRanked, InstalledInstanceID: "brainstorming", WeaponVersion: "1.0.0"}).Ref())
	assert.Equal(t, "brainstorming@1.0.0", (SlotBinding{Mode: SlotBindingModeCustom, InstalledInstanceID: "brainstorming@1.0.0", WeaponVersion: "1.0.0"}).Ref())
}

func TestWeaponContractAndIdentityValidationCoverage(t *testing.T) {
	valid := WeaponContract{RoleOwner: "ranger", Participation: "required", InvocationEvidence: "required", UnavailableBehavior: "role_invocation_failed", NativeSubstitution: "forbidden"}
	require.NoError(t, valid.Validate())
	require.NoError(t, (WeaponContract{}).Validate())
	for name, mutate := range map[string]func(*WeaponContract){
		"role":          func(w *WeaponContract) { w.RoleOwner = "" },
		"participation": func(w *WeaponContract) { w.Participation = "optional" },
		"evidence":      func(w *WeaponContract) { w.InvocationEvidence = "optional" },
		"unavailable":   func(w *WeaponContract) { w.UnavailableBehavior = "fallback" },
		"substitution":  func(w *WeaponContract) { w.NativeSubstitution = "allowed" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			require.Error(t, candidate.Validate())
		})
	}

	require.NoError(t, (CanonicalIdentity{Family: TaxonomyWeapon, ID: "brainstorming", Version: "1.0.0"}).Validate())
	require.Error(t, (CanonicalIdentity{Family: TaxonomyWeapon, ID: "brainstorming"}).Validate())
	require.Error(t, (CanonicalIdentity{Family: TaxonomyRole}).Validate())
	assert.Equal(t, CanonicalIdentity{Family: TaxonomyWeapon, ID: "w", Version: "1.0.0"}, (CompiledWeapon{ID: "w", Version: "1.0.0"}).CanonicalIdentity())
}

func TestRoleSourceArtifactValidationErrorCoverage(t *testing.T) {
	valid, err := NewRoleSourceArtifact("ranger", "1", "1.0.0", "", strings.Repeat("a", 64), strings.Repeat("b", 64))
	require.NoError(t, err)
	for name, mutate := range map[string]func(*RoleSourceArtifact){
		"invalid role":   func(a *RoleSourceArtifact) { a.Role = "unknown" },
		"skill path":     func(a *RoleSourceArtifact) { a.SkillManifestPath = "skill.yaml" },
		"skill digest":   func(a *RoleSourceArtifact) { a.SkillManifestDigest = "bad" },
		"skill identity": func(a *RoleSourceArtifact) { a.SkillVersion = "" },
		"schema":         func(a *RoleSourceArtifact) { a.SchemaVersion = "wrong" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			require.Error(t, candidate.Validate())
		})
	}
}

func TestRoleLoadoutValidationErrorCoverage(t *testing.T) {
	resolution, err := ResolveStage(StageResolutionRequest{Route: MissionRouteFullPipeline, Role: "ranger"})
	require.NoError(t, err)
	weapon := RoleInvocationPlan{
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0",
		WeaponDigest: "sha256:weapon", BindingDigest: "sha256:binding",
		Runtime: WeaponRuntime{Kind: RankedRuntimeEmbedded},
	}
	base := func() RoleLoadout {
		return RoleLoadout{
			SchemaVersion: RoleLoadoutSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion,
			Resolution: resolution, Role: "ranger", Slot: "discovery", Weapon: weapon,
			Feats: []LoadoutCapability{{Family: TaxonomyFeat, ID: "initiative", Availability: CapabilityAvailable}},
			Tools: []LoadoutCapability{{Family: TaxonomyTool, ID: "leveling", Availability: CapabilityAvailable}},
		}
	}
	for name, mutate := range map[string]func(*RoleLoadout){
		"schema":             func(l *RoleLoadout) { l.SchemaVersion = "wrong" },
		"taxonomy":           func(l *RoleLoadout) { l.TaxonomyVersion = "wrong" },
		"resolution":         func(l *RoleLoadout) { l.Resolution = StageResolution{} },
		"roster stage":       func(l *RoleLoadout) { l.Resolution.Stage = StageRoster },
		"missing identity":   func(l *RoleLoadout) { l.Role = "" },
		"role mismatch":      func(l *RoleLoadout) { l.Resolution.Role = "sniper" },
		"missing digest":     func(l *RoleLoadout) { l.Weapon.WeaponDigest = "" },
		"duplicate tool":     func(l *RoleLoadout) { l.Tools = append(l.Tools, l.Tools[0]) },
		"wrong family":       func(l *RoleLoadout) { l.Feats[0].Family = TaxonomyTool },
		"missing capability": func(l *RoleLoadout) { l.Feats[0].ID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			loadout := base()
			mutate(&loadout)
			require.Error(t, loadout.Validate())
		})
	}
}

func TestWeaponRuntimeValidationErrorCoverage(t *testing.T) {
	valid := []WeaponRuntime{
		{Kind: RankedRuntimeNone},
		{Kind: RankedRuntimeHost, HostAPI: "host/v1"},
		{Kind: RankedRuntimeEmbedded},
		{Kind: RankedRuntimeExecutable, Entrypoint: "run"},
		{Kind: RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "boot", Healthcheck: "health", Version: "1.2.3", NodeVersion: "20.19.0"},
		{Kind: RankedRuntimeEmbedded, ExecutionMode: WeaponExecutionModePromptBridge},
	}
	for _, runtime := range valid {
		require.NoError(t, runtime.Validate())
	}
	for name, runtime := range map[string]WeaponRuntime{
		"prompt bridge host":            {Kind: RankedRuntimeHost, ExecutionMode: WeaponExecutionModePromptBridge},
		"host missing api":              {Kind: RankedRuntimeHost},
		"host extra field":              {Kind: RankedRuntimeHost, HostAPI: "host", Entrypoint: "bad"},
		"executable missing entrypoint": {Kind: RankedRuntimeExecutable},
		"executable extra field":        {Kind: RankedRuntimeExecutable, Entrypoint: "run", HostAPI: "bad"},
		"openspec unsafe root":          {Kind: RankedRuntimeOpenSpecRoot, Root: "/tmp", Bootstrap: "boot", Healthcheck: "health"},
		"openspec missing healthcheck":  {Kind: RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "boot"},
		"openspec invalid version":      {Kind: RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "boot", Healthcheck: "health", Version: "latest"},
		"unknown kind":                  {Kind: "future"},
	} {
		t.Run(name, func(t *testing.T) { require.Error(t, runtime.Validate()) })
	}
}

func TestConfidenceReportAndSummaryCalibrationCoverage(t *testing.T) {
	calibrated := ConfidenceSummary{
		PolicyVersion:     ConfidencePolicyVersion,
		SampleSize:        CalibrationMinimumSample,
		CalibrationStatus: CalibrationCalibrated,
		Claims: []ConfidenceClaim{{
			ID: "calibrated", Statement: "resolved", Agent: "ranger", CorrelationKey: "calibrated",
			ClaimKind: ClaimKindQuestion, ConfidencePercent: 80,
			GroundTruthRef: "handoff:1", GroundTruthKind: GroundTruthHandoff, GroundTruthOutcome: GroundTruthCorrect,
		}},
	}
	require.NoError(t, ValidateConfidenceSummary(calibrated))
	artifact, err := ConfidenceReportFromSummary(calibrated, "report", "ranger", "run", "provider")
	require.NoError(t, err)
	require.NoError(t, artifact.Validate())

	for _, level := range []string{ConfidenceLow, ConfidenceMedium, ConfidenceHigh} {
		candidate := artifact
		candidate.ConfidenceLevel = level
		require.NoError(t, candidate.Validate())
	}
	invalid := artifact
	invalid.ConfidenceLevel = "unknown"
	require.ErrorContains(t, invalid.Validate(), "unsupported confidence_level")

	for _, candidate := range []ConfidenceReportArtifact{
		{SchemaVersion: ConfidenceReportArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Status: "unknown"},
		{SchemaVersion: ConfidenceReportArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Status: ConfidenceReportAvailable, ReportID: "id", Source: "src"},
		{SchemaVersion: ConfidenceReportArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Status: ConfidenceReportMissing},
		{SchemaVersion: ConfidenceReportArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Status: ConfidenceReportIncompatible},
	} {
		require.Error(t, candidate.Validate())
	}
}

func TestEffortAndBindingArtifactValidationCoverage(t *testing.T) {
	for _, effort := range []string{"none", "low", "medium", "high", "xhigh", "max"} {
		artifact := EffortResolutionArtifact{SchemaVersion: EffortResolutionArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Role: "ranger", Effort: effort, Source: "host"}
		require.NoError(t, artifact.Validate())
	}
	base := EffortResolutionArtifact{SchemaVersion: EffortResolutionArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Role: "ranger"}
	for _, candidate := range []EffortResolutionArtifact{
		{SchemaVersion: "bad", TaxonomyVersion: CanonicalTaxonomyVersion, Role: "ranger"},
		{SchemaVersion: EffortResolutionArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion},
		{SchemaVersion: EffortResolutionArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Role: "ranger", Effort: "bad"},
		{SchemaVersion: EffortResolutionArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Role: "ranger", Source: "bad"},
		{SchemaVersion: EffortResolutionArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Role: "ranger", PolicyVersion: -1},
		{SchemaVersion: EffortResolutionArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Role: "ranger", FallbackUsed: true},
	} {
		require.Error(t, candidate.Validate())
	}
	base.PolicyVersion, base.PolicyDigest, base.FallbackUsed, base.FallbackReason = 1, "sha256:policy", true, "provider unavailable"
	require.NoError(t, base.Validate())

	valid := coverageCompleteBinding("discovery", "ranger", "brainstorming", "1.0.0")
	artifacts, err := NewWeaponBindingArtifacts([]SlotBinding{valid})
	require.NoError(t, err)
	require.NoError(t, ValidateWeaponBindingArtifacts([]SlotBinding{valid}, artifacts))
	require.NoError(t, ValidateWeaponBindingArtifacts(nil, nil))
	require.ErrorContains(t, ValidateWeaponBindingArtifacts(nil, artifacts), "without bindings")
	require.ErrorContains(t, ValidateWeaponBindingArtifacts([]SlotBinding{valid, valid}, artifacts), "duplicate slot")
	assert.ErrorContains(t, ValidateWeaponBindingArtifacts([]SlotBinding{valid}, []WeaponBindingArtifact{artifacts[0], artifacts[0]}), "artifact count")
}

func TestAdditionalDomainValidationBranchesCoverage(t *testing.T) {
	assert.NotEqual(t, 0, CompareWeaponVersions("1.bad", "1.0"))
	assert.Equal(t, `ranked runtime state schema "v1" is not "`+RankedRuntimeStateSchemaVersion+`"`, (&RankedRuntimeStateLegacyError{SchemaVersion: "v1"}).Error())
	state := RankedRuntimeState{Entries: []RankedRuntimeStateEntry{{Slot: "discovery", Provider: "p"}}}
	_, ok := state.Entry("missing", "p")
	assert.False(t, ok)
	runtime := &RankedRuntimeStateRuntime{Components: []RankedRuntimeStateComponent{{Name: "node"}}}
	_, ok = runtime.Component("openspec")
	assert.False(t, ok)

	request := MissionInvocationRequest{
		Protocol: MissionInvocationProtocolVersion, RequestID: "request", MissionID: "mission", Role: "ranger", Slot: "discovery",
		Weapon: MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"}, BindingDigest: "sha256:binding", SourceDigest: "sha256:source",
		ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload",
	}
	for _, candidate := range []MissionInvocationRequest{
		{},
		func() MissionInvocationRequest { candidate := request; candidate.Protocol = "old"; return candidate }(),
	} {
		require.Error(t, candidate.Validate())
	}
	require.ErrorContains(t, (MissionInvocationCompletion{}).Validate(), "request_id")

	base := WeaponManifest{ID: "suite", Version: "1.0.0", Kind: WeaponKindComposite, Origin: WeaponOriginCustom, Roles: []string{"ranger"}, SupportedSlots: []string{"discovery"}, Runtime: WeaponRuntime{Kind: WeaponRuntimeHost, HostAPI: "strategist-host-skill/v1"}}
	base.Composition = &WeaponComposition{Strategy: WeaponCompositionOrderedDAG, Components: []WeaponComponent{{ID: "brainstorming", Required: true}}}
	require.NoError(t, base.Validate())
	atomic := base
	atomic.Kind, atomic.Composition = WeaponKindAtomic, base.Composition
	require.ErrorContains(t, atomic.Validate(), "must not declare composition")
	invalid := base
	invalid.Roles = []string{"ranger", "ranger"}
	require.ErrorContains(t, invalid.Validate(), "duplicate role")
	invalid = base
	invalid.ID, invalid.Version, invalid.Kind = "", "", "future"
	require.Error(t, invalid.Validate())

	binding := SlotBinding{Role: "ranger", Slot: "discovery", InstalledInstanceID: "brainstorming@1.0.0", WeaponVersion: "1.0.0"}
	require.ErrorContains(t, ValidateRoleWeaponIdentity(CompiledRegistry{}, "", "discovery", binding), "requires id")
	require.ErrorContains(t, ValidateRoleWeaponIdentity(CompiledRegistry{}, "ranger", "", binding), "slot is required")
	binding.Role = "archivist"
	require.ErrorContains(t, ValidateRoleWeaponIdentity(CompiledRegistry{}, "ranger", "discovery", binding), "does not match")
	binding.Role, binding.Slot = "ranger", "refinement"
	assert.ErrorContains(t, ValidateRoleWeaponIdentity(CompiledRegistry{}, "ranger", "discovery", binding), "binding slot")
}

func coverageCompleteBinding(slot, role, weapon, version string) SlotBinding {
	return SlotBinding{
		SchemaVersion: "strategist-plugin-binding/v1", Slot: slot, InstalledInstanceID: weapon + "@" + version,
		Role: role, WeaponVersion: version, WeaponDigest: "sha256:weapon-" + weapon,
		SourceDigest: "sha256:source-" + weapon, BindingDigest: "sha256:binding-" + weapon,
		RuntimeKind: "host", ConnectorID: "host", Entrypoint: "host.prompt", Origin: string(WeaponOriginCustom),
		Status: "active", Generation: 1,
	}
}
