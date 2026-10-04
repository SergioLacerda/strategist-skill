package domain

import "fmt"

func validateMissionStatus(status MissionEngineStatus) error {
	if err := validateMissionIdentity(status); err != nil {
		return err
	}
	if err := validateStageMetadata(status); err != nil {
		return err
	}
	return validateMissionPhaseState(status)
}

func validateStageMetadata(status MissionEngineStatus) error {
	if status.Stage == "" {
		return validateAbsentStageMetadata(status)
	}
	if err := status.Stage.Validate(); err != nil {
		return fmt.Errorf("mission engine: %w", err)
	}
	if status.Stage != StageShort || status.StageFeat != "critical_hit" {
		return fmt.Errorf("mission engine: unsupported mission Stage metadata")
	}
	return validateCriticalHitStageMetadata(status)
}

func validateAbsentStageMetadata(status MissionEngineStatus) error {
	if status.StageFeat != "" || status.StageCorrelationID != "" || status.StageGateRequired || status.StageGateApproved {
		return fmt.Errorf("mission engine: Stage metadata is incomplete")
	}
	return nil
}

func validateCriticalHitStageMetadata(status MissionEngineStatus) error {
	if status.StageCorrelationID == "" || !status.StageGateRequired {
		return fmt.Errorf("mission engine: Critical Hit Stage metadata requires correlation and gate")
	}
	if status.StageGateApproved && status.State != StateExecution && status.State != StateDoneDelivery {
		return fmt.Errorf("mission engine: approved Critical Hit Stage requires execution or completion state")
	}
	return nil
}

func validateMissionIdentity(status MissionEngineStatus) error {
	if status.MissionID == "" {
		return fmt.Errorf("mission engine: mission id is required")
	}
	if status.Phase == "" || status.State == "" {
		return fmt.Errorf("mission engine: phase and state are required")
	}
	return nil
}

func validateMissionPhaseState(status MissionEngineStatus) error {
	if err := validateHandoffMetadata(status); err != nil {
		return err
	}
	if status.Phase == PhaseBlocked && status.State != StateBlocked {
		return fmt.Errorf("mission engine: blocked phase requires blocked state")
	}
	if status.State == StateInit && !earlyMissionPhase(status.Phase) {
		return fmt.Errorf("mission engine: INIT state requires an early pipeline phase")
	}
	if !validMissionState(status.Phase, status.State) {
		return fmt.Errorf("mission engine: state %q is invalid for phase %q", status.State, status.Phase)
	}
	return nil
}

func validateHandoffMetadata(status MissionEngineStatus) error {
	if status.HandoffAttempt < 0 {
		return fmt.Errorf("mission engine: handoff attempt cannot be negative")
	}
	if status.HandoffAttempt == 0 && (status.HandoffStatus != "" || status.HandoffNextAction != "") {
		return fmt.Errorf("mission engine: handoff metadata requires a positive attempt")
	}
	return nil
}

func validMissionState(phase PipelinePhase, state MissionState) bool {
	valid := map[PipelinePhase]map[MissionState]bool{
		PhaseBootstrap: {StateInit: true}, PhaseIntake: {StateInit: true}, PhaseDiscovery: {StateInit: true},
		PhaseRefinement:   {StateRefinement: true, StateRetryingRefinement: true},
		PhaseApprovalGate: {StateApprovalGate: true, StateHandoffChallenge: true, StateSideQuestGate: true, StateADRGate1: true, StateADRGate2: true, StateDirectGate: true},
		PhaseExecution:    {StateExecution: true, StateSideQuestExec: true, StateRetryingExecution: true, StateDirectExec: true, StateRetryingDirectExec: true},
		PhaseDone:         {StateDoneAnalysis: true, StateDoneDelivery: true, StateADRDone: true, StateDirectDone: true},
		PhaseBlocked:      {StateBlocked: true},
	}
	states, ok := valid[phase]
	return ok && states[state]
}

func earlyMissionPhase(phase PipelinePhase) bool {
	return phase == PhaseBootstrap || phase == PhaseIntake || phase == PhaseDiscovery
}
