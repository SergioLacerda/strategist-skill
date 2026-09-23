package initiative

import (
	"fmt"
	"path/filepath"
)

// LedgerRelPath is the independent runtime history for INITIATIVE records.
// It is deliberately different from LEVELING's role-levels.jsonl path.
const LedgerRelPath = "memory/initiative-records.jsonl"

// LedgerPath returns the INITIATIVE ledger path for a Strategist runtime root.
func LedgerPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(LedgerRelPath))
}

// Runtime is the internal role-boundary producer for INITIATIVE. It owns only
// advisory advice/result records; it never reads or writes LEVELING records.
type Runtime struct {
	Advisor    Advisor
	LedgerFile string
}

// NewRuntime creates a runtime backed by the independent INITIATIVE ledger.
func NewRuntime(strategistRoot string, policy Policy) (Runtime, error) {
	if err := policy.Validate(); err != nil {
		return Runtime{}, err
	}
	return Runtime{Advisor: Advisor{Policy: policy}, LedgerFile: LedgerPath(strategistRoot)}, nil
}

// EnterRole resolves one initial advice envelope for a role run. If an
// initial advice already exists for the same mission/role/run, it is reused
// instead of creating a competing record.
func (r Runtime) EnterRole(input AdviceInput) (advice Advice, reused bool, err error) {
	if err := r.validate(); err != nil {
		return Advice{}, false, err
	}
	if input.Trigger != TriggerInitial {
		return Advice{}, false, fmt.Errorf("initiative runtime: role entry requires initial trigger")
	}
	err = withLedgerLock(r.LedgerFile, func() error { return r.enterRoleLocked(input, &advice, &reused) })
	if err != nil {
		return Advice{}, false, err
	}
	return advice, reused, nil
}

// Reevaluate creates one explicit superseding advice record. The previous
// advice is discovered from the independent ledger and remains immutable.
func (r Runtime) Reevaluate(input AdviceInput) (Advice, error) {
	if err := r.validate(); err != nil {
		return Advice{}, err
	}
	if err := validateReevaluationInput(r.Advisor.Policy, input); err != nil {
		return Advice{}, err
	}
	var advice Advice
	err := withLedgerLock(r.LedgerFile, func() error {
		return r.reevaluateLocked(input, &advice)
	})
	if err != nil {
		return Advice{}, err
	}
	return advice, nil
}

func validateReevaluationInput(policy Policy, input AdviceInput) error {
	if input.Trigger == TriggerInitial {
		return fmt.Errorf("initiative runtime: re-evaluation requires a non-initial trigger")
	}
	if !policy.AllowsTrigger(input.Trigger) {
		return fmt.Errorf("initiative runtime: trigger %q is not enabled by policy", input.Trigger)
	}
	return nil
}

func (r Runtime) reevaluateLocked(input AdviceInput, advice *Advice) error {
	previous, found, err := LatestAdvice(r.LedgerFile, input.MissionID, input.Role, input.RunID)
	if err != nil {
		return err
	}
	if !found || previous.Advice == nil {
		return fmt.Errorf("initiative runtime: cannot re-evaluate without prior advice")
	}
	if input.Supersedes != "" && input.Supersedes != previous.AdviceID {
		return fmt.Errorf("initiative runtime: supersedes %q does not match latest advice %q", input.Supersedes, previous.AdviceID)
	}
	input.Supersedes = previous.AdviceID
	sequence, err := adviceSequence(r.LedgerFile, input.MissionID, input.Role, input.RunID)
	if err != nil {
		return err
	}
	input.Sequence = sequence + 1
	*advice, err = r.Advisor.Advise(input)
	if err != nil {
		return err
	}
	return appendRecordUnlocked(r.LedgerFile, Record{
		Kind: RecordKindAdvice, MissionID: advice.MissionID, Role: advice.Role,
		RunID: advice.RunID, AdviceID: advice.AdviceID, Supersedes: advice.Supersedes,
		Advice: advice,
	})
}

// RecordResult validates and appends one role result. The returned assessment
// is advisory: callers must not use it as Approval Gate or LEVELING authority.
func (r Runtime) RecordResult(advice Advice, result Result) (ResultAssessment, error) {
	if err := r.validate(); err != nil {
		return ResultAssessment{}, err
	}
	var assessment ResultAssessment
	err := withLedgerLock(r.LedgerFile, func() error { return r.recordResultLocked(advice, result, &assessment) })
	if err != nil {
		return ResultAssessment{}, err
	}
	return assessment, nil
}

func (r Runtime) enterRoleLocked(input AdviceInput, advice *Advice, reused *bool) error {
	existing, found, err := LatestAdvice(r.LedgerFile, input.MissionID, input.Role, input.RunID)
	if err != nil {
		return err
	}
	if found && existing.Advice != nil {
		*advice = *existing.Advice
		*reused = true
		return nil
	}
	*advice, err = r.Advisor.Advise(input)
	if err != nil {
		return err
	}
	return appendRecordUnlocked(r.LedgerFile, Record{
		Kind: RecordKindAdvice, MissionID: advice.MissionID, Role: advice.Role,
		RunID: advice.RunID, AdviceID: advice.AdviceID, Advice: advice,
	})
}

func (r Runtime) recordResultLocked(advice Advice, result Result, assessment *ResultAssessment) error {
	latest, found, err := LatestAdvice(r.LedgerFile, advice.MissionID, advice.Role, advice.RunID)
	if err != nil {
		return err
	}
	if !found || latest.Advice == nil {
		return fmt.Errorf("initiative runtime: cannot record result without persisted advice")
	}
	if err := validatePersistedAdvice(advice, *latest.Advice, r.Advisor.Policy); err != nil {
		return err
	}
	*assessment, err = AssessResult(advice, result)
	if err != nil {
		return err
	}
	return appendRecordUnlocked(r.LedgerFile, Record{
		Kind: RecordKindResult, MissionID: result.MissionID, Role: result.Role,
		RunID: result.RunID, AdviceID: result.AdviceID, Result: &result,
	})
}
