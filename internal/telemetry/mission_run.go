package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type missionRunKey struct{}

// MissionRun tracks end-to-end mission timing and visible output volume.
type MissionRun struct {
	mu            sync.Mutex
	MissionID     string
	startedAt     time.Time
	intakeAt      time.Time
	scoutAt       time.Time
	rangerAt      time.Time
	archivistAt   time.Time
	gateAt        time.Time
	gateRespondAt time.Time
	sniperAt      time.Time
	linesOut      int64
	tokensIn      int64
	tokensOut     int64
	silent        bool // when true, Finish skips EmitMissionMetrics
}

// NewMissionRun initializes a mission tracker.
func NewMissionRun(missionID string) *MissionRun {
	return &MissionRun{
		MissionID: missionID,
		startedAt: time.Now(),
	}
}

// WithMissionRun stores the tracker in the context.
func WithMissionRun(ctx context.Context, run *MissionRun) context.Context {
	return context.WithValue(ctx, missionRunKey{}, run)
}

// MissionRunFromContext extracts the tracker from the context.
func MissionRunFromContext(ctx context.Context) *MissionRun {
	if ctx == nil {
		return nil
	}
	run, ok := ctx.Value(missionRunKey{}).(*MissionRun)
	if !ok {
		return nil
	}
	return run
}

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

// AddLines increments the visible output counter.
func (m *MissionRun) AddLines(n int64) {
	if n <= 0 {
		return
	}
	m.mu.Lock()
	m.linesOut += n
	m.mu.Unlock()
}

// SetTokens records token counts when they are known.
func (m *MissionRun) SetTokens(in, out int64) {
	m.mu.Lock()
	m.tokensIn = in
	m.tokensOut = out
	m.mu.Unlock()
}

// Snapshot returns the current metrics values.
func (m *MissionRun) Snapshot() MissionMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	intakeAt := firstNonZeroTime(m.intakeAt, now)
	scoutAt := firstNonZeroTime(m.scoutAt, intakeAt)
	rangerAt := firstNonZeroTime(m.rangerAt, scoutAt)
	archivistAt := firstNonZeroTime(m.archivistAt, rangerAt)
	gateAt := firstNonZeroTime(m.gateAt, archivistAt)
	gateRespondAt := firstNonZeroTime(m.gateRespondAt, gateAt)
	sniperAt := firstNonZeroTime(m.sniperAt, gateRespondAt)

	return MissionMetrics{
		MissionID:            m.MissionID,
		TStartToIntakeMS:     intakeAt.Sub(m.startedAt).Milliseconds(),
		TIntakeToScoutMS:     scoutAt.Sub(intakeAt).Milliseconds(),
		TScoutToRangerMS:     rangerAt.Sub(scoutAt).Milliseconds(),
		TIntakeToRangerMS:    rangerAt.Sub(intakeAt).Milliseconds(),
		TRangerToArchivistMS: archivistAt.Sub(rangerAt).Milliseconds(),
		TArchivistToGateMS:   gateAt.Sub(archivistAt).Milliseconds(),
		TGateWaitMS:          gateRespondAt.Sub(gateAt).Milliseconds(),
		TGateToSniperMS:      sniperAt.Sub(gateRespondAt).Milliseconds(),
		TSniperToDoneMS:      now.Sub(sniperAt).Milliseconds(),
		TotalWallTimeMS:      now.Sub(m.startedAt).Milliseconds(),
		TokensIn:             m.tokensIn,
		TokensOut:            m.tokensOut,
		LinesEmitted:         m.linesOut,
	}
}

func firstNonZeroTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}

// SetSilent suppresses the metrics emission in Finish. Call this for
// interactive/wizard commands where all-zero metrics add no signal.
func (m *MissionRun) SetSilent() {
	m.mu.Lock()
	m.silent = true
	m.mu.Unlock()
}

// Finish emits the current metrics snapshot (unless SetSilent was called).
func (m *MissionRun) Finish() {
	m.mu.Lock()
	silent := m.silent
	m.mu.Unlock()
	if silent {
		return
	}
	m.AddLines(1)
	EmitMissionMetrics(m.Snapshot())
}

// FinishMission emits metrics for the mission attached to ctx, if any.
func FinishMission(ctx context.Context) {
	if run := MissionRunFromContext(ctx); run != nil {
		run.Finish()
	}
}

// StartLine returns a canonical pipeline-starting line for this mission.
func (m *MissionRun) StartLine(profileMode, profilePath, activeYAMLPath, personaResolved, reason, outputProfile string) string {
	return fmt.Sprintf(
		"[Strategist] pipeline=starting mission_id=%s profile_mode=%s profile_path=%s active_yaml_path=%s persona_resolved=%s reason=%s output=%s",
		m.MissionID, profileMode, SanitizePath(profilePath), SanitizePath(activeYAMLPath), personaResolved, reason, outputProfile,
	)
}
