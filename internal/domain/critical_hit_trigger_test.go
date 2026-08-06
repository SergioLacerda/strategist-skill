package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func validPlainMoveEvidence() domain.CriticalHitEvidence {
	return domain.CriticalHitEvidence{
		Mode:       domain.CriticalHitModePlain,
		TaskType:   "analysis_move",
		SourcePath: ".analysis/pending/foo-analysis.md",
		TargetPath: ".analysis/archived/foo-analysis.md",
		BasePath:   ".analysis",
		FileTypes:  []string{".md"},
		RiskLevel:  "low",
		FileCount:  1,
	}
}

func validClosureMoveEvidence() domain.CriticalHitEvidence {
	return domain.CriticalHitEvidence{
		Mode:                    domain.CriticalHitModeClosure,
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
	decision := domain.EvaluateCriticalHit(validPlainMoveEvidence())

	assert.True(t, decision.Allowed)
	assert.Equal(t, domain.CriticalHitModePlain, decision.Mode)
	assert.Empty(t, decision.Reason)
	assert.Empty(t, decision.FallbackRoute)
}

func TestEvaluateCriticalHit_AllowsValidClosureMove(t *testing.T) {
	t.Parallel()
	decision := domain.EvaluateCriticalHit(validClosureMoveEvidence())

	assert.True(t, decision.Allowed)
	assert.Equal(t, domain.CriticalHitModeClosure, decision.Mode)
}

func TestEvaluateCriticalHit_PlainMove_BlocksWrongTaskType(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.TaskType = "refactor"

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "conditions_not_met", decision.Reason)
	assert.Equal(t, "main_mission", decision.FallbackRoute)
}

func TestEvaluateCriticalHit_PlainMove_BlocksSourceOutsideAnalysisFolders(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.SourcePath = "internal/domain/critical_hit_trigger.go"

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksTargetOutsideBasePath(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.TargetPath = "/etc/passwd"

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksNonMarkdownFileType(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.FileTypes = []string{".go"}

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksHighRisk(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.RiskLevel = "high"

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksTooManyFiles(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.FileCount = 6

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_PlainMove_BlocksExplicitCompletionClaim(t *testing.T) {
	t.Parallel()
	e := validPlainMoveEvidence()
	e.ExplicitCompletionClaim = true

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksSourceOutsidePendingOrRefined(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.SourcePath = ".analysis/archived/foo"

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksTargetOutsideDone(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.TargetPath = ".analysis/archived/foo"

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksMissingCompletionClaim(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.ExplicitCompletionClaim = false

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksMissingEvidenceSummary(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.EvidenceSummaryPresent = false

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksCompletionInferredFromCodeOnly(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.CompletionInferredFromCodeOnly = true

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}

func TestEvaluateCriticalHit_ClosureMove_BlocksPartialImplementationWithResiduals(t *testing.T) {
	t.Parallel()
	e := validClosureMoveEvidence()
	e.PartialImplementationWithDeclaredResiduals = true

	decision := domain.EvaluateCriticalHit(e)

	assert.False(t, decision.Allowed)
}
