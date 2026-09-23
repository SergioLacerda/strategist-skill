package initiative

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func (r Runtime) validate() error {
	if err := r.Advisor.Policy.Validate(); err != nil {
		return err
	}
	if r.LedgerFile == "" {
		return fmt.Errorf("initiative runtime: ledger path is required")
	}
	return nil
}

func adviceSequence(path, missionID, role, runID string) (int, error) {
	records, err := ReadRecords(path)
	if err != nil {
		return 0, err
	}
	sequence := 0
	for _, record := range records {
		if record.Kind == RecordKindAdvice && record.MissionID == missionID && record.Role == normalizeRole(role) && record.RunID == runID {
			sequence++
		}
	}
	return sequence, nil
}

func validatePersistedAdvice(candidate, persisted Advice, policy Policy) error {
	if candidate.PolicyVersion != policy.Version || candidate.PolicyDigest != policy.Digest() {
		return fmt.Errorf("initiative runtime: advice policy identity does not match active policy")
	}
	candidateJSON, err := json.Marshal(candidate)
	if err != nil {
		return fmt.Errorf("initiative runtime: encode advice: %w", err)
	}
	persistedJSON, err := json.Marshal(persisted)
	if err != nil {
		return fmt.Errorf("initiative runtime: encode persisted advice: %w", err)
	}
	if !bytes.Equal(candidateJSON, persistedJSON) {
		return fmt.Errorf("initiative runtime: advice does not match persisted latest advice")
	}
	return nil
}
