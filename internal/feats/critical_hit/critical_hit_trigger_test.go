package criticalhit_test

import (
	"testing"

	criticalhit "github.com/SergioLacerda/strategist-skill/internal/feats/critical_hit"
	"github.com/stretchr/testify/assert"
)

func validPlainMoveEvidence() criticalhit.Evidence {
	return criticalhit.Evidence{
		Mode:       criticalhit.ModePlain,
		TaskType:   "analysis_move",
		SourcePath: ".analysis/pending/foo-analysis.md",
		TargetPath: ".analysis/archived/foo-analysis.md",
		BasePath:   ".analysis",
		FileTypes:  []string{".md"},
		RiskLevel:  "low",
		FileCount:  1,
	}
}

func validClosureMoveEvidence() criticalhit.Evidence {
	return criticalhit.Evidence{
		Mode:                    criticalhit.ModeClosure,
		TaskType:                "analysis_move",
		SourcePath:              ".analysis/refined/foo",
		TargetPath:              ".analysis/done/foo",
		BasePath:                ".analysis",
		ExplicitCompletionClaim: true,
		EvidenceSummaryPresent:  true,
	}
}

func TestEvaluateCriticalHit_AllowsValidPlainMove(t *testing.T) {
	t.Parallel()
	decision := criticalhit.EvaluateCriticalHit(validPlainMoveEvidence())

	assert.True(t, decision.Allowed)
	assert.Equal(t, criticalhit.ModePlain, decision.Mode)
	assert.Empty(t, decision.Reason)
	assert.Empty(t, decision.FallbackRoute)
}

func TestEvaluateCriticalHit_AllowsValidClosureMove(t *testing.T) {
	t.Parallel()
	decision := criticalhit.EvaluateCriticalHit(validClosureMoveEvidence())

	assert.True(t, decision.Allowed)
	assert.Equal(t, criticalhit.ModeClosure, decision.Mode)
}

func TestEvaluateCriticalHit_PlainMove_BlocksWrongTaskType(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.TaskType = "refactor"

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "conditions_not_met", decision.Reason)
	assert.Equal(t, "full_pipeline", decision.FallbackRoute)
}

func TestEvaluateCriticalHit_PlainMove_BlocksSourceOutsideAnalysisFolders(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.SourcePath = "internal/domain/critical_hit_trigger.go"

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksTargetOutsideBasePath(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.TargetPath = "/etc/passwd"

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksNonMarkdownFileType(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.FileTypes = []string{".go"}

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksHighRisk(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.RiskLevel = "high"

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksTooManyFiles(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.FileCount = 6

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksExplicitCompletionClaim(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.ExplicitCompletionClaim = true

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksSourceOutsidePendingOrRefined(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.SourcePath = ".analysis/archived/foo"

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksTargetOutsideDone(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.TargetPath = ".analysis/archived/foo"

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksMissingCompletionClaim(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.ExplicitCompletionClaim = false

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksMissingEvidenceSummary(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.EvidenceSummaryPresent = false

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksCompletionInferredFromCodeOnly(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.CompletionInferredFromCodeOnly = true

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksPartialImplementationWithResiduals(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.PartialImplementationWithDeclaredResiduals = true

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_BlocksUnknownMode(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.Mode = "unknown"

	decision := criticalhit.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
	assert.Equal(t, criticalhit.Mode("unknown"), decision.Mode)
	assert.Equal(t, "conditions_not_met", decision.Reason)
	assert.Equal(t, "full_pipeline", decision.FallbackRoute)
}
