package initiative

import "fmt"

func validateRecord(record Record) error {
	if !validRecordKind(record.Kind) {
		return fmt.Errorf("initiative_record_invalid: unknown kind %q", record.Kind)
	}
	if !hasRecordIdentity(record) {
		return fmt.Errorf("initiative_record_invalid: mission, role, run, and advice identity are required")
	}
	if record.Kind == RecordKindAdvice {
		return validateAdviceRecord(record)
	}
	return validateResultRecord(record)
}

func validRecordKind(kind string) bool {
	return kind == RecordKindAdvice || kind == RecordKindResult
}

func hasRecordIdentity(record Record) bool {
	return record.MissionID != "" && normalizeRole(record.Role) != "" &&
		record.RunID != "" && record.AdviceID != ""
}

func validateAdviceRecord(record Record) error {
	if record.Advice == nil || record.Result != nil || record.Advice.AdviceID != record.AdviceID {
		return fmt.Errorf("initiative_record_invalid: advice record payload mismatch")
	}
	return record.Advice.Validate()
}

func validateResultRecord(record Record) error {
	if record.Result == nil || record.Advice != nil || record.Result.AdviceID != record.AdviceID {
		return fmt.Errorf("initiative_record_invalid: result record payload mismatch")
	}
	if record.Result.ResultID == "" || record.Result.Sequence < 1 {
		return fmt.Errorf("initiative_record_invalid: result revision identity and positive sequence are required")
	}
	if record.Assessment == nil {
		return nil
	}
	return validateRecordAssessment(record)
}

func validateRecordAssessment(record Record) error {
	if !assessmentMatchesRecord(*record.Assessment, record) {
		return fmt.Errorf("initiative_record_invalid: assessment payload mismatch")
	}
	return validateAssessment(*record.Assessment)
}

func assessmentMatchesRecord(assessment ResultAssessment, record Record) bool {
	return assessment.AssessmentID != "" && assessment.InputDigest != "" &&
		assessment.Mechanism == PreciseShotMechanism && assessment.SchemaVersion != "" &&
		assessment.AdviceID == record.AdviceID && assessment.ResultID == record.Result.ResultID &&
		assessment.ResultSequence == record.Result.Sequence
}
