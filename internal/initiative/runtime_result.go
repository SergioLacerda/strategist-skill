package initiative

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func (r Runtime) recordResultLocked(advice Advice, result Result, assessment *ResultAssessment) error {
	if err := r.requirePersistedAdvice(advice); err != nil {
		return err
	}
	result, err := r.resolveResultRevision(advice, result)
	if err != nil {
		return err
	}
	if *assessment, err = AssessResult(advice, result); err != nil {
		return err
	}
	if err := r.boundEscalation(advice, assessment); err != nil {
		return err
	}
	return appendRecordUnlocked(r.LedgerFile, Record{
		Kind: RecordKindResult, MissionID: result.MissionID, Role: result.Role,
		RunID: result.RunID, AdviceID: result.AdviceID, Supersedes: result.Supersedes, Result: &result, Assessment: assessment,
	})
}

// requirePersistedAdvice fails unless the ledger holds advice matching the
// supplied one under the active policy.
func (r Runtime) requirePersistedAdvice(advice Advice) error {
	latest, found, err := LatestAdvice(r.LedgerFile, advice.MissionID, advice.Role, advice.RunID)
	if err != nil {
		return err
	}
	if !found || latest.Advice == nil {
		return fmt.Errorf("initiative runtime: cannot record result without persisted advice")
	}
	return validatePersistedAdvice(advice, *latest.Advice, r.Advisor.Policy)
}

// resolveResultRevision checks the result against the current ledger head and
// fills the sequence and result id of a first result.
func (r Runtime) resolveResultRevision(advice Advice, result Result) (Result, error) {
	previous, found, err := LatestResult(r.LedgerFile, advice.MissionID, advice.Role, advice.RunID, advice.AdviceID)
	if err != nil {
		return result, err
	}
	if found {
		return result, checkResultRevision(previous, result)
	}
	return firstResult(advice, result)
}

func checkResultRevision(previous, result Result) error {
	if result.ResultID == "" || result.Supersedes != previous.ResultID || result.Sequence != previous.Sequence+1 {
		return fmt.Errorf("initiative_result_revision_conflict: result must supersede current result %q at sequence %d", previous.ResultID, previous.Sequence+1)
	}
	return nil
}

func firstResult(advice Advice, result Result) (Result, error) {
	if result.Sequence == 0 {
		result.Sequence = 1
	}
	if result.Sequence != 1 || result.Supersedes != "" {
		return result, fmt.Errorf("initiative_result_revision_conflict: first result must start at sequence 1 without supersession")
	}
	if result.ResultID == "" {
		result.ResultID = resultID(advice, result.Sequence)
	}
	return result, nil
}

// boundEscalation applies the persisted escalation budget to an escalation
// request carried by the assessment.
func (r Runtime) boundEscalation(advice Advice, assessment *ResultAssessment) error {
	if assessment.Escalation == nil {
		return nil
	}
	state, err := persistedEscalationState(r.LedgerFile, advice.MissionID, advice.Role, advice.RunID)
	if err != nil {
		return err
	}
	state.MaximumEffort = advice.Observed.Effort == EffortMax || advice.Observed.Effort == EffortXHigh
	bounded, err := BoundEscalation(*assessment.Escalation, state, DefaultEscalationPolicy())
	if err != nil {
		return err
	}
	assessment.Escalation = &bounded
	return nil
}

func persistedEscalationState(path, missionID, role, runID string) (EscalationState, error) {
	records, err := ReadRecords(path)
	if err != nil {
		return EscalationState{}, err
	}
	state := EscalationState{}
	for _, record := range records {
		if !escalationRecordFor(record, missionID, role, runID) {
			continue
		}
		applyEscalationRecord(&state, record)
	}
	return state, nil
}

// escalationRecordFor reports whether record is a result of this role run that
// carries an escalation request.
func escalationRecordFor(record Record, missionID, role, runID string) bool {
	return record.Kind == RecordKindResult && record.Assessment != nil && record.Assessment.Escalation != nil &&
		record.MissionID == missionID && record.Role == normalizeRole(role) && record.RunID == runID
}

func applyEscalationRecord(state *EscalationState, record Record) {
	escalation := record.Assessment.Escalation
	if escalation.Status == EscalationRequested {
		state.RequestCount++
	}
	if record.Result != nil && record.Result.Sequence >= state.LastResultSequence {
		state.LastResultSequence = record.Result.Sequence
		state.LastEffective = record.Assessment.Effective
		state.LastAdviceID = record.Assessment.AdviceID
		state.LastIdempotencyKey = escalation.IdempotencyKey
	}
}

func resultID(advice Advice, sequence int) string {
	material := fmt.Sprintf("%s\x00%s\x00%s\x00%d", advice.AdviceID, advice.MissionID, advice.RunID, sequence)
	sum := sha256.Sum256([]byte(material))
	return "res-" + hex.EncodeToString(sum[:])[:24]
}
