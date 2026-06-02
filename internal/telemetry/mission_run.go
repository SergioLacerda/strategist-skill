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
	mu        sync.Mutex
	MissionID string
	startedAt time.Time
	intakeAt  time.Time
	rangerAt  time.Time
	linesOut  int64
	tokensIn  int64
	tokensOut int64
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
	if m.intakeAt.IsZero() {
		m.intakeAt = time.Now()
	}
}

// MarkRanger records the first substantive work timestamp once.
func (m *MissionRun) MarkRanger() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rangerAt.IsZero() {
		m.rangerAt = time.Now()
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
	intakeAt := m.intakeAt
	if intakeAt.IsZero() {
		intakeAt = now
	}
	rangerAt := m.rangerAt
	if rangerAt.IsZero() {
		rangerAt = intakeAt
	}

	return MissionMetrics{
		MissionID:         m.MissionID,
		TStartToIntakeMS:  intakeAt.Sub(m.startedAt).Milliseconds(),
		TIntakeToRangerMS: rangerAt.Sub(intakeAt).Milliseconds(),
		TotalWallTimeMS:   now.Sub(m.startedAt).Milliseconds(),
		TokensIn:          m.tokensIn,
		TokensOut:         m.tokensOut,
		LinesEmitted:      m.linesOut,
	}
}

// Finish emits the current metrics snapshot.
func (m *MissionRun) Finish() {
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
		m.MissionID, profileMode, profilePath, activeYAMLPath, personaResolved, reason, outputProfile,
	)
}
