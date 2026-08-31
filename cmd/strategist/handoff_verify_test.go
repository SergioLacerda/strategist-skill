package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errorWriter always fails writes, for exercising output-error branches.
// internal/check carries its own duplicate (Go test helpers aren't
// shareable across package boundaries) since the treasure-chest precedent
// (20260806-treasure-chest-cmd-consolidation) established the pattern.
type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("forced write failure")
}

const handoffVerifyChallengesYAML = `
challenges:
  - id: HC-001
    type: objective
    source_refs: [G-001]
    critical: true
  - id: HC-002
    type: boundary
    source_refs: [X-001]
    critical: true
  - id: HC-003
    type: classification
    source_refs: [D-001, Q-001]
    critical: true
    expected_classification:
      D-001: approved_decision
      Q-001: unresolved_question
  - id: HC-004
    type: gate
    source_refs: [approval.required]
    critical: true
    expected_gate_allowed: false
  - id: HC-005
    type: counterfactual
    source_refs: [C-01]
    critical: true
    expected_counterfactual: false
`

const handoffVerifyAckPassYAML = `
challenge_refs: [HC-001, HC-002, HC-003, HC-004, HC-005]
understood_refs: [G-001, X-001, D-001, Q-001, approval.required, C-01]
classifications:
  D-001: approved_decision
  Q-001: unresolved_question
gate_allowed: false
counterfactual_answers:
  HC-005: false
`

const handoffVerifyAckFailYAML = `
challenge_refs: [HC-001, HC-002, HC-003, HC-004, HC-005]
understood_refs: [G-001, X-001, D-001, Q-001, approval.required, C-01]
classifications:
  D-001: approved_decision
  Q-001: approved_decision
gate_allowed: true
counterfactual_answers:
  HC-005: true
`

func setHandoffVerifyFlags(t *testing.T, root, transition, policy, challenges, ack, missionID string, attempt int) {
	t.Helper()
	require.NoError(t, handoffVerifyCmd.Flags().Set(flagRoot, root))
	require.NoError(t, handoffVerifyCmd.Flags().Set("transition", transition))
	require.NoError(t, handoffVerifyCmd.Flags().Set("policy", policy))
	require.NoError(t, handoffVerifyCmd.Flags().Set("challenges", challenges))
	require.NoError(t, handoffVerifyCmd.Flags().Set("ack", ack))
	require.NoError(t, handoffVerifyCmd.Flags().Set("mission-id", missionID))
	require.NoError(t, handoffVerifyCmd.Flags().Set("attempt", strconv.Itoa(attempt)))
}

func resetHandoffVerifyFlags(t *testing.T) {
	t.Helper()
	setHandoffVerifyFlags(t, "", "", "", "", "", "", 1)
	require.NoError(t, handoffVerifyCmd.Flags().Set("risk-level", ""))
}

// setHandoffVerifyRiskLevel sets --risk-level in isolation, for tests
// exercising handoff.ResolvePolicyForMission wiring. Kept separate from
// setHandoffVerifyFlags so existing call sites don't all need a new
// parameter for a flag most of them don't care about.
func setHandoffVerifyRiskLevel(t *testing.T, level string) {
	t.Helper()
	require.NoError(t, handoffVerifyCmd.Flags().Set("risk-level", level))
}

func writeHandoffVerifyFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestHandoffVerifyCmd_PassesAndRecords(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	challenges := writeHandoffVerifyFixture(t, dir, "challenges.yaml", handoffVerifyChallengesYAML)
	ack := writeHandoffVerifyFixture(t, dir, "ack.yaml", handoffVerifyAckPassYAML)
	setHandoffVerifyFlags(t, dir, "archivist_to_sniper", "", challenges, ack, "m-pass", 1)
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	out := captureStdout(t, func() {
		require.NoError(t, handoffVerifyCmd.RunE(handoffVerifyCmd, nil))
	})

	assert.Contains(t, out, "status: passed")
	assert.Contains(t, out, "passed: true")

	data, err := os.ReadFile(filepath.Join(dir, "memory", "handoff-challenges.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"mission_id":"m-pass"`)
	assert.Contains(t, string(data), `"passed":true`)
}

func TestHandoffVerifyCmd_FailsAndStillRecords(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	challenges := writeHandoffVerifyFixture(t, dir, "challenges.yaml", handoffVerifyChallengesYAML)
	ack := writeHandoffVerifyFixture(t, dir, "ack.yaml", handoffVerifyAckFailYAML)
	setHandoffVerifyFlags(t, dir, "archivist_to_sniper", "", challenges, ack, "m-fail", 2)
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	var runErr error
	out := captureStdout(t, func() {
		runErr = handoffVerifyCmd.RunE(handoffVerifyCmd, nil)
	})

	require.Error(t, runErr)
	assert.Contains(t, out, "status: failed")
	assert.Contains(t, out, "misclassified_refs")
	assert.Contains(t, out, "counterfactual_mismatches")
	assert.Contains(t, out, "gate_mismatch: true")

	data, err := os.ReadFile(filepath.Join(dir, "memory", "handoff-challenges.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"mission_id":"m-fail"`)
	assert.Contains(t, string(data), `"passed":false`)
}

func TestHandoffVerifyCmd_UnknownTransitionErrorsWithoutPolicy(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	challenges := writeHandoffVerifyFixture(t, dir, "challenges.yaml", handoffVerifyChallengesYAML)
	ack := writeHandoffVerifyFixture(t, dir, "ack.yaml", handoffVerifyAckPassYAML)
	setHandoffVerifyFlags(t, dir, "not_a_real_transition", "", challenges, ack, "m-bad", 1)
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	err := handoffVerifyCmd.RunE(handoffVerifyCmd, nil)
	require.ErrorContains(t, err, "unknown --transition")
}

func TestHandoffVerifyCmd_SniperToValidationTransitionPasses(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	challenges := writeHandoffVerifyFixture(t, dir, "challenges.yaml", `
challenges:
  - id: HC-201
    type: boundary
    source_refs: [FILE-001]
    critical: true
  - id: HC-202
    type: classification
    source_refs: [DEV-001]
    critical: true
    expected_classification:
      DEV-001: authorized_deviation
`)
	ack := writeHandoffVerifyFixture(t, dir, "ack.yaml", `
understood_refs: [FILE-001, DEV-001]
classifications:
  DEV-001: authorized_deviation
`)
	setHandoffVerifyFlags(t, dir, "sniper_to_validation", "", challenges, ack, "m-validation", 1)
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	out := captureStdout(t, func() {
		require.NoError(t, handoffVerifyCmd.RunE(handoffVerifyCmd, nil))
	})
	assert.Contains(t, out, "status: passed")
}

func TestHandoffVerifyCmd_LowRiskLevelSkipsRangerToArchivist(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	// Deliberately empty/incomplete challenges and ack: if risk-based
	// resolution failed to disable the policy, Verify would fail on these
	// (missing required challenge types and refs) instead of skipping.
	challenges := writeHandoffVerifyFixture(t, dir, "challenges.yaml", "challenges: []\n")
	ack := writeHandoffVerifyFixture(t, dir, "ack.yaml", "understood_refs: []\n")
	setHandoffVerifyFlags(t, dir, "ranger_to_archivist", "", challenges, ack, "m-low-risk", 1)
	setHandoffVerifyRiskLevel(t, "low")
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	out := captureStdout(t, func() {
		require.NoError(t, handoffVerifyCmd.RunE(handoffVerifyCmd, nil))
	})
	assert.Contains(t, out, "status: skipped")
	assert.Contains(t, out, "passed: true")
}

func TestHandoffVerifyCmd_HighRiskLevelRequiresRangerToArchivist(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	// Same deliberately incomplete fixtures as the low-risk case above:
	// with the policy enabled by --risk-level=high, these same inputs must
	// now fail verification instead of being skipped.
	challenges := writeHandoffVerifyFixture(t, dir, "challenges.yaml", "challenges: []\n")
	ack := writeHandoffVerifyFixture(t, dir, "ack.yaml", "understood_refs: []\n")
	setHandoffVerifyFlags(t, dir, "ranger_to_archivist", "", challenges, ack, "m-high-risk", 1)
	setHandoffVerifyRiskLevel(t, "high")
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	var runErr error
	out := captureStdout(t, func() {
		runErr = handoffVerifyCmd.RunE(handoffVerifyCmd, nil)
	})
	require.Error(t, runErr)
	assert.Contains(t, out, "status: failed")
	assert.Contains(t, out, "missing_challenges")
}

func TestHandoffVerifyCmd_HighRiskLevelPassesSniperToValidationWithFullAck(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	challenges := writeHandoffVerifyFixture(t, dir, "challenges.yaml", `
challenges:
  - id: HC-301
    type: boundary
    source_refs: [FILE-001]
    critical: true
  - id: HC-302
    type: classification
    source_refs: [DEV-001]
    critical: true
    expected_classification:
      DEV-001: authorized_deviation
`)
	ack := writeHandoffVerifyFixture(t, dir, "ack.yaml", `
understood_refs: [FILE-001, DEV-001]
classifications:
  DEV-001: authorized_deviation
`)
	setHandoffVerifyFlags(t, dir, "sniper_to_validation", "", challenges, ack, "m-high-risk-validation", 1)
	setHandoffVerifyRiskLevel(t, "high")
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	out := captureStdout(t, func() {
		require.NoError(t, handoffVerifyCmd.RunE(handoffVerifyCmd, nil))
	})
	assert.Contains(t, out, "status: passed")
}

func TestHandoffVerifyCmd_RiskLevelUnknownTransitionErrors(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	challenges := writeHandoffVerifyFixture(t, dir, "challenges.yaml", "challenges: []\n")
	ack := writeHandoffVerifyFixture(t, dir, "ack.yaml", "understood_refs: []\n")
	setHandoffVerifyFlags(t, dir, "not_a_real_transition", "", challenges, ack, "m-bad", 1)
	setHandoffVerifyRiskLevel(t, "high")
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	err := handoffVerifyCmd.RunE(handoffVerifyCmd, nil)
	require.ErrorContains(t, err, "unknown transition")
}

func TestHandoffVerifyCmd_PolicyFileOverridesTransition(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	challenges := writeHandoffVerifyFixture(t, dir, "challenges.yaml", handoffVerifyChallengesYAML)
	ack := writeHandoffVerifyFixture(t, dir, "ack.yaml", handoffVerifyAckPassYAML)
	policy := writeHandoffVerifyFixture(t, dir, "policy.yaml", `
enabled: true
transition: archivist_to_sniper
required_types: [objective, boundary, classification, gate, counterfactual]
require_all_critical: true
max_attempts: 2
on_failure: return_to_archivist
forbidden_claims:
  - execution_authorized
`)
	setHandoffVerifyFlags(t, dir, "", policy, challenges, ack, "m-policy", 1)
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	out := captureStdout(t, func() {
		require.NoError(t, handoffVerifyCmd.RunE(handoffVerifyCmd, nil))
	})
	assert.Contains(t, out, "status: passed")
}

func TestHandoffVerifyCmd_MissingRequiredFlags(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setHandoffVerifyFlags(t, dir, "archivist_to_sniper", "", "", "", "", 1)
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	err := handoffVerifyCmd.RunE(handoffVerifyCmd, nil)
	require.ErrorContains(t, err, "--challenges")
	require.ErrorContains(t, err, "--ack")
	require.ErrorContains(t, err, "--mission-id")
}

func TestHandoffCmd_IsHumanStatusCommand(t *testing.T) {
	assert.True(t, isHumanStatusCommand(handoffVerifyCmd))
}

func TestLoadHandoffPolicy_ReadError(t *testing.T) {
	_, err := loadHandoffPolicy(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	require.ErrorContains(t, err, "read policy file")
}

func TestLoadHandoffPolicy_ParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.yaml")
	require.NoError(t, os.WriteFile(path, []byte("not: [valid\n"), 0o644))

	_, err := loadHandoffPolicy(path)
	require.ErrorContains(t, err, "parse policy file")
}

func TestLoadHandoffChallenges_ReadError(t *testing.T) {
	_, err := loadHandoffChallenges(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	require.ErrorContains(t, err, "read challenges file")
}

func TestLoadHandoffChallenges_ParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "challenges.yaml")
	require.NoError(t, os.WriteFile(path, []byte("not: [valid\n"), 0o644))

	_, err := loadHandoffChallenges(path)
	require.ErrorContains(t, err, "parse challenges file")
}

func TestLoadHandoffAck_ReadError(t *testing.T) {
	_, err := loadHandoffAck(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	require.ErrorContains(t, err, "read acknowledgment file")
}

func TestLoadHandoffAck_ParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ack.yaml")
	require.NoError(t, os.WriteFile(path, []byte("not: [valid\n"), 0o644))

	_, err := loadHandoffAck(path)
	require.ErrorContains(t, err, "parse acknowledgment file")
}

func TestPrintHandoffVerifyResult_WriteError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{})

	err := printHandoffVerifyResult(cmd, handoff.Result{Status: "passed", Passed: true})
	require.ErrorContains(t, err, "write output")
}

func TestRecordHandoffVerify_AppendErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	setHandoffVerifyFlags(t, blocker, "archivist_to_sniper", "", "", "", "m-1", 1)
	t.Cleanup(func() { resetHandoffVerifyFlags(t) })

	err := recordHandoffVerify(handoffVerifyCmd, handoffVerifyOptions{Root: blocker, MissionID: "m-1", Attempt: 1}, handoff.Result{Status: "passed", Passed: true})
	require.ErrorContains(t, err, "record handoff challenge")
}
