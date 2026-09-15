package telemetry

import "time"

// MarkIntake records the intake timestamp once.
func (m *MissionRun) MarkIntake() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markTimeOnce(&m.intakeAt)
}

// MarkScout records when Scout's route decision completes, once.
func (m *MissionRun) MarkScout() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markTimeOnce(&m.scoutAt)
}

// MarkRanger records the first substantive work timestamp once.
func (m *MissionRun) MarkRanger() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markTimeOnce(&m.rangerAt)
}

// MarkArchivist records when the refinement slot starts.
func (m *MissionRun) MarkArchivist() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markTimeOnce(&m.archivistAt)
}

// MarkGatePresented records when the approval gate is shown to the user.
func (m *MissionRun) MarkGatePresented() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markTimeOnce(&m.gateAt)
}

// MarkGateResponse records when the user responds to the approval gate.
// TGateWaitMS = MarkGateResponse − MarkGatePresented (human latency).
func (m *MissionRun) MarkGateResponse() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markTimeOnce(&m.gateRespondAt)
}

// MarkSniper records when the execution slot starts.
func (m *MissionRun) MarkSniper() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markTimeOnce(&m.sniperAt)
}

func (m *MissionRun) markTimeOnce(dst *time.Time) {
	if dst.IsZero() {
		*dst = time.Now()
	}
}
