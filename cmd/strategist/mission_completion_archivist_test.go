package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	catalogadapter "github.com/SergioLacerda/strategist-skill/internal/catalog"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const archivistMissionID = "arch-mission"

const archivistPending = `---
schema_version: strategist-ranger-discovery/v1
mission_id: arch-mission
mission_status: ranger_done
sources_consulted: []
ranger_handoff_policy_facts:
  schema_version: strategist-ranger-handoff-policy-facts/v1
  require_recall: false
  require_boundary: false
  require_classification: false
  require_verdict: false
  informational_only: true
discovery_subtype: creative
---

## mission_objective
objective
## known_facts
- id: F-1
  statement: fact
## confidence_summary
summary
## handoff
handoff
`

const archivistResult = `---
handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: []
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: true
---

## proposal.md

### Why

reason

## design.md

### Context

context

## tasks.md

- [ ] 1.1 [task_type: analysis_artifact] record the evidence and verify the note exists
`

// archivistWorkspace builds a hermetic root whose refinement slot is bound to
// the certified Ranked embedded writing-plans Weapon, with a pending Ranger
// analysis whose typed facts let the Ranger-to-Archivist handoff skip.
func archivistWorkspace(t *testing.T, authorize bool) (string, string) {
	t.Helper()
	workspace := t.TempDir()
	root, basePath := filepath.Join(workspace, ".strategist"), filepath.Join(workspace, ".analysis")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o755))
	catalog, err := (strategistembed.Extractor{}).ReadFile("plugins/catalog.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), catalog, 0o600))
	registry, err := catalogadapter.ParseCompiledRegistryCatalog(catalog)
	require.NoError(t, err)
	offer, ok := registry.RankedBinding("archivist", "refinement", "writing-plans", "1.0.0")
	require.True(t, ok, "writing-plans is a certified Ranked Archivist offer")
	binding := domain.SlotBinding{
		SchemaVersion: "strategist-plugin-binding/v1", Slot: "refinement", InstalledInstanceID: offer.WeaponID,
		Role: "archivist", WeaponVersion: offer.WeaponVersion, WeaponDigest: offer.WeaponDigest, SourceDigest: offer.SourceDigest,
		BindingDigest: offer.BindingDigest, ExecutionMode: offer.ExecutionMode, Origin: string(domain.WeaponOriginEmbedded),
		RuntimeKind: offer.Runtime.Kind, ConnectorID: offer.ConnectorID, Entrypoint: offer.Entrypoint,
		CertificationDigest: offer.CertificationDigest, Status: offer.Status, Mode: domain.SlotBindingModeRanked,
	}
	writeYAML(t, filepath.Join(root, "active.yaml"), domain.ActiveConfig{Slots: map[string]string{"refinement": offer.WeaponID}})
	writeYAML(t, filepath.Join(root, "plugins.lock"), domain.PluginLockFile{Bindings: []domain.SlotBinding{binding}})
	status, err := json.Marshal(domain.MissionEngineStatus{MissionID: archivistMissionID, Phase: domain.PhaseRefinement})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", archivistMissionID+".json"), status, 0o600))
	pending := filepath.Join(basePath, "pending", archivistMissionID+"-analysis.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.WriteFile(pending, []byte(archivistPending), 0o600))
	if authorize {
		_, err := missionruntime.EvaluateRangerToArchivist(root, pending, archivistMissionID, missionruntime.RangerHandoffInput{})
		require.NoError(t, err)
	}
	return root, basePath
}

func archivistBuildInput(root, basePath string) missionadapter.InvocationBuildInput {
	return missionadapter.InvocationBuildInput{Root: root, BasePath: basePath, MissionID: archivistMissionID, Role: "archivist", Slot: "refinement"}
}

func archivistComplete(root, basePath, requestID, result string) missionadapter.InvocationCompleteInput {
	return missionadapter.InvocationCompleteInput{
		Root: root, BasePath: basePath, RequestID: requestID, Adapter: domain.ExecutionAdapterCurrentHost,
		Completion: domain.MissionInvocationCompletion{RequestID: requestID, Result: result},
		Sink:       &captureSink{},
	}
}

func TestBuildMissionInvocationIssuesAnArchivistEnvelopeForTheRankedEmbeddedWeapon(t *testing.T) {
	root, basePath := archivistWorkspace(t, true)

	request, err := buildMissionInvocation(t.Context(), archivistBuildInput(root, basePath))

	require.NoError(t, err)
	assert.Equal(t, "writing-plans", request.Weapon.ID)
	assert.Equal(t, "prompt_bridge", request.ExecutionMode)
	assert.Contains(t, request.Payload, "writing-plans")
	assert.Contains(t, request.Input["execution_contract"], "Archivist refinement")
	assert.Contains(t, request.Input["execution_contract"], "Do not ask the user questions", "the interactive hand-off of the payload is not part of the adapter")
	assert.Contains(t, request.Input["output_contract"], "## proposal.md")
	assert.Contains(t, request.Input["output_contract"], "handoff_policy_facts")
	assert.Equal(t, filepath.ToSlash(filepath.Join(basePath, "pending", archivistMissionID+"-analysis.md")), request.Input["analysis_artifact_path"])
}

func TestBuildMissionInvocationDeniesTheArchivistWithoutAnAuthorizingRangerOutcome(t *testing.T) {
	root, basePath := archivistWorkspace(t, false)

	_, err := buildMissionInvocation(t.Context(), archivistBuildInput(root, basePath))

	require.ErrorContains(t, err, "role_invocation_failed")
	entries, readErr := os.ReadDir(filepath.Join(root, "missions", "invocations"))
	if readErr == nil {
		assert.Empty(t, entries, "a denied entry persists no request")
	}
}

func TestCompleteMissionInvocationPublishesTheArchivistPackageAndRejectsAReplay(t *testing.T) {
	root, basePath := archivistWorkspace(t, true)
	request, err := buildMissionInvocation(t.Context(), archivistBuildInput(root, basePath))
	require.NoError(t, err)

	outcome, err := completeMissionInvocation(t.Context(), archivistComplete(root, basePath, request.RequestID, archivistResult))

	require.NoError(t, err)
	assert.Equal(t, "normalized", outcome.Status)
	assert.Equal(t, filepath.ToSlash(filepath.Join(".analysis", "refined", archivistMissionID)), outcome.ArtifactPath)
	refined := filepath.Join(basePath, "refined", archivistMissionID)
	for _, name := range []string{"analysis.md", "proposal.md", "design.md", "tasks.md"} {
		assert.FileExists(t, filepath.Join(refined, name))
	}
	analysis, err := os.ReadFile(filepath.Join(refined, "analysis.md"))
	require.NoError(t, err)
	assert.Contains(t, string(analysis), "mission_status: archivist_done")
	assert.Contains(t, string(analysis), "provider: writing-plans")
	assert.NoFileExists(t, filepath.Join(basePath, "pending", archivistMissionID+"-analysis.md"), "the pending analysis is promoted")

	_, err = completeMissionInvocation(t.Context(), archivistComplete(root, basePath, request.RequestID, archivistResult))
	require.ErrorContains(t, err, "invocation_replay")
}

func TestCompleteMissionInvocationNeverPublishesAMalformedArchivistCompletion(t *testing.T) {
	root, basePath := archivistWorkspace(t, true)
	request, err := buildMissionInvocation(t.Context(), archivistBuildInput(root, basePath))
	require.NoError(t, err)
	discoveryShaped := "## mission_objective\nx\n## known_facts\n- f\n## confidence_summary\nc\n## handoff\nh\n"

	_, err = completeMissionInvocation(t.Context(), archivistComplete(root, basePath, request.RequestID, discoveryShaped))

	require.Error(t, err)
	assert.NoDirExists(t, filepath.Join(basePath, "refined", archivistMissionID))
	pending, readErr := os.ReadFile(filepath.Join(basePath, "pending", archivistMissionID+"-analysis.md"))
	require.NoError(t, readErr)
	assert.Equal(t, archivistPending, string(pending), "a discovery-shaped result never reaches the discovery normalizer for an Archivist request")

	retry, err := completeMissionInvocation(t.Context(), archivistComplete(root, basePath, request.RequestID, archivistResult))
	require.NoError(t, err, "a rejected completion leaves the request completable")
	assert.Equal(t, "normalized", retry.Status)
}

func TestCompleteMissionInvocationRejectsAnUnclassifiedArchivistTaskBeforeCommitting(t *testing.T) {
	root, basePath := archivistWorkspace(t, true)
	request, err := buildMissionInvocation(t.Context(), archivistBuildInput(root, basePath))
	require.NoError(t, err)
	unclassified := "- [ ] 1.1 [task_type: analysis_artifact] record the evidence and verify the note exists"

	_, err = completeMissionInvocation(t.Context(), archivistComplete(root, basePath, request.RequestID,
		replaceOnce(t, archivistResult, unclassified, "- [ ] 1.1 record the evidence and verify the note exists")))

	require.ErrorContains(t, err, "classification")
	assert.NoDirExists(t, filepath.Join(basePath, "refined", archivistMissionID))
}

func replaceOnce(t *testing.T, text, old, replacement string) string {
	t.Helper()
	require.Contains(t, text, old)
	return strings.Replace(text, old, replacement, 1)
}
