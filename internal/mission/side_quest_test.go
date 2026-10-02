package mission

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

type sqFixture struct {
	workspace, root, base string
}

func newSQFixture(t *testing.T, missionID string) sqFixture {
	t.Helper()
	ws := t.TempDir()
	f := sqFixture{workspace: ws, root: filepath.Join(ws, ".strategist"), base: filepath.Join(ws, ".analysis")}
	pkg := filepath.Join(f.base, "refined", missionID)
	if err := os.MkdirAll(pkg, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(f.root, 0o750); err != nil {
		t.Fatal(err)
	}
	tasks := "# t\n```yaml\nside_quests_approved:\n  - {id: OA-ADR-" + missionID + ", description: d, strategy: execute_together, status: sq_pending}\n```\n"
	if err := os.WriteFile(filepath.Join(pkg, "tasks.md"), []byte(tasks), 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f sqFixture) decide(missionID, decision, canonical string, state domain.MissionState) (AcceptedSideQuest, error) {
	return DecideSideQuest(SideQuestDecision{
		StrategistRoot: f.root, BasePath: f.base, MissionID: missionID, SideQuestID: OAADRSideQuestID(missionID),
		Decision: decision, State: state, CanonicalPath: canonical, Now: time.Now(),
	})
}

func (f sqFixture) target(missionID string, state domain.MissionState) (string, error) {
	return ReserveADRTarget(ADRTargetRequest{StrategistRoot: f.root, BasePath: f.base, MissionID: missionID, State: state, Now: time.Now()})
}

func (f sqFixture) sealed(t *testing.T, missionID, canonical string) {
	t.Helper()
	if _, err := f.decide(missionID, SideQuestAccepted, canonical, domain.StateApprovalGate); err != nil {
		t.Fatal(err)
	}
	if err := SealSideQuestOnGateApproval(f.root, missionID, domain.MissionEventGateApproved, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestDecideSideQuestRoundTripAndIdempotence(t *testing.T) {
	f := newSQFixture(t, "m1")
	rec, err := f.decide("m1", SideQuestAccepted, "docs/adr", domain.StateApprovalGate)
	if err != nil {
		t.Fatal(err)
	}
	loaded, found, err := LoadAcceptedSideQuest(f.root, "m1")
	if err != nil || !found || loaded != rec || loaded.Destination.CanonicalPath != "docs/adr" || loaded.Destination.Fallback != "archived/m1-adr.md" {
		t.Fatalf("round trip: %v %v %+v", err, found, loaded)
	}
	again, err := f.decide("m1", SideQuestAccepted, "docs/adr", domain.StateApprovalGate)
	if err != nil || again != rec {
		t.Fatalf("idempotent re-accept: %v", err)
	}
}

func TestDecideSideQuestRejectionsLeaveNoRecord(t *testing.T) {
	f := newSQFixture(t, "m1")
	cases := map[string]func() error{
		"wrong state": func() error { _, err := f.decide("m1", SideQuestAccepted, "", domain.StateExecution); return err },
		"foreign id": func() error {
			_, err := DecideSideQuest(SideQuestDecision{StrategistRoot: f.root, BasePath: f.base, MissionID: "m1", SideQuestID: "OA-ADR-other", Decision: SideQuestAccepted, State: domain.StateApprovalGate})
			return err
		},
		"absent from package": func() error {
			g := newSQFixture(t, "m2")
			_ = os.Remove(filepath.Join(g.base, "refined", "m2", "tasks.md"))
			_, err := g.decide("m2", SideQuestAccepted, "", domain.StateApprovalGate)
			return err
		},
		"absolute destination": func() error {
			_, err := f.decide("m1", SideQuestAccepted, "/etc", domain.StateApprovalGate)
			return err
		},
		"escaping destination": func() error {
			_, err := f.decide("m1", SideQuestAccepted, "../adr", domain.StateApprovalGate)
			return err
		},
		"bad decision": func() error { _, err := f.decide("m1", "maybe", "", domain.StateApprovalGate); return err },
	}
	for name, run := range cases {
		if err := run(); err == nil {
			t.Errorf("%s: want rejection", name)
		}
	}
	if _, found, err := LoadAcceptedSideQuest(f.root, "m1"); err != nil || found {
		t.Fatalf("rejections must not write a record: %v %v", err, found)
	}
}

func TestSideQuestDecisionExclusivityAndSealing(t *testing.T) {
	f := newSQFixture(t, "m1")
	if _, err := f.decide("m1", SideQuestDeclined, "", domain.StateApprovalGate); err != nil {
		t.Fatal(err)
	}
	if _, err := f.decide("m1", SideQuestAccepted, "", domain.StateApprovalGate); err != nil {
		t.Fatalf("an unsealed decision may change: %v", err)
	}
	if err := SealSideQuestOnGateApproval(f.root, "m1", domain.MissionEventGateApproved, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.decide("m1", SideQuestDeclined, "", domain.StateApprovalGate); err == nil || !strings.Contains(err.Error(), "side_quest_record_sealed") {
		t.Fatalf("a sealed decision must not change: %v", err)
	}
	if _, err := f.decide("m1", SideQuestAccepted, "", domain.StateApprovalGate); err != nil {
		t.Fatalf("re-accepting a sealed acceptance is idempotent: %v", err)
	}
}

func TestSideQuestRecordTamperIsDetected(t *testing.T) {
	f := newSQFixture(t, "m1")
	if _, err := f.decide("m1", SideQuestAccepted, "", domain.StateApprovalGate); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.root, "missions", "side-quests", "m1.json")
	raw, err := os.ReadFile(path) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"accepted"`, `"declined"`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadAcceptedSideQuest(f.root, "m1"); err == nil || !strings.Contains(err.Error(), "side_quest_record_tampered") {
		t.Fatalf("tamper must be detected: %v", err)
	}
	if err := RequireNoAcceptedSideQuest(f.root, "m1", domain.MissionEventGateApprovedAnalysisOnly); err == nil {
		t.Fatal("a tampered record must fail closed")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func wantErr(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), code) {
		t.Fatalf("want an error containing %q, got %v", code, err)
	}
}

// touch creates dir and an empty file for each name inside it.
func touch(t *testing.T, dir string, names ...string) {
	t.Helper()
	must(t, os.MkdirAll(dir, 0o750))
	for _, name := range names {
		must(t, os.WriteFile(filepath.Join(dir, name), nil, 0o600))
	}
}

func (f sqFixture) mustDecide(t *testing.T, missionID, decision, canonical string) {
	t.Helper()
	_, err := f.decide(missionID, decision, canonical, domain.StateApprovalGate)
	must(t, err)
}

func (f sqFixture) mustTarget(t *testing.T, missionID string) string {
	t.Helper()
	got, err := f.target(missionID, domain.StateExecution)
	must(t, err)
	return got
}

func (f sqFixture) claimedPaths(t *testing.T) []string {
	t.Helper()
	claims, err := telemetry.ReadRecentSniperClaims(telemetry.SniperClaimHistoryPath(f.root), time.Now(), time.Hour)
	must(t, err)
	paths := make([]string, 0, len(claims))
	for _, c := range claims {
		paths = append(paths, c.TargetPath)
	}
	return paths
}

var analysisOnlyEvents = []domain.MissionEngineEvent{domain.MissionEventGateApprovedAnalysisOnly, domain.MissionEventHandoffNotApplicable}

func TestGateEventGuardWithoutARecordBehavesAsBefore(t *testing.T) {
	f := newSQFixture(t, "m1")
	for _, evt := range analysisOnlyEvents {
		must(t, RequireNoAcceptedSideQuest(f.root, "m1", evt))
	}
}

func TestGateEventGuardRejectsAnalysisOnlyEventsWhenAccepted(t *testing.T) {
	f := newSQFixture(t, "m1")
	f.mustDecide(t, "m1", SideQuestAccepted, "")
	for _, evt := range analysisOnlyEvents {
		if err := RequireNoAcceptedSideQuest(f.root, "m1", evt); !errors.Is(err, ErrGateEventConflictsWithSideQuest) {
			t.Fatalf("%s must be rejected: %v", evt, err)
		}
	}
	must(t, RequireNoAcceptedSideQuest(f.root, "m1", domain.MissionEventGateApproved))
}

func TestGateEventGuardLeavesAnalysisOnlyAvailableWhenDeclined(t *testing.T) {
	f := newSQFixture(t, "m1")
	f.mustDecide(t, "m1", SideQuestDeclined, "")
	for _, evt := range analysisOnlyEvents {
		must(t, RequireNoAcceptedSideQuest(f.root, "m1", evt))
	}
}

func TestSealOnlyOnGateApprovedAndOnlyWithRecord(t *testing.T) {
	f := newSQFixture(t, "m1")
	if err := SealSideQuestOnGateApproval(f.root, "m1", domain.MissionEventGateApproved, time.Now()); err != nil {
		t.Fatalf("no record: %v", err)
	}
	if _, err := f.decide("m1", SideQuestAccepted, "", domain.StateApprovalGate); err != nil {
		t.Fatal(err)
	}
	if err := SealSideQuestOnGateApproval(f.root, "m1", domain.MissionEventGateDenied, time.Now()); err != nil {
		t.Fatal(err)
	}
	if rec, _, _ := LoadAcceptedSideQuest(f.root, "m1"); rec.Sealed {
		t.Fatal("only gate_approved seals")
	}
}

func TestReserveADRTargetCanonicalNumbering(t *testing.T) {
	f := newSQFixture(t, "m1")
	touch(t, filepath.Join(f.workspace, "docs", "adr"), "0007-a.md", "0012-b.md", "notes.md")
	f.sealed(t, "m1", "docs/adr")
	if got := f.mustTarget(t, "m1"); got != "docs/adr/0013-m1.md" {
		t.Fatalf("canonical numbering: %q", got)
	}
}

func TestReserveADRTargetReplayKeepsOneReservationAndOneClaim(t *testing.T) {
	f := newSQFixture(t, "m1")
	dir := filepath.Join(f.workspace, "docs", "adr")
	touch(t, dir, "0007-a.md")
	f.sealed(t, "m1", "docs/adr")
	got := f.mustTarget(t, "m1")
	touch(t, dir, filepath.Base(got)) // Sniper wrote the file; a replay must not renumber.
	if again := f.mustTarget(t, "m1"); again != got {
		t.Fatalf("replay must return the reserved path: %q != %q", again, got)
	}
	if claims := f.claimedPaths(t); len(claims) != 1 || claims[0] != got {
		t.Fatalf("exactly one claim for the reserved path: %v", claims)
	}
}

func TestReserveADRTargetFallsBackToTheArchivedPath(t *testing.T) {
	f := newSQFixture(t, "m1")
	f.sealed(t, "m1", "")
	if got := f.mustTarget(t, "m1"); got != ".analysis/archived/m1-adr.md" {
		t.Fatalf("archived fallback: %q", got)
	}
}

func TestReserveADRTargetWithoutANumberedConventionUsesTheMissionName(t *testing.T) {
	f := newSQFixture(t, "m2")
	touch(t, filepath.Join(f.workspace, "docs", "adr"))
	f.sealed(t, "m2", "docs/adr")
	if got := f.mustTarget(t, "m2"); got != "docs/adr/m2-adr.md" {
		t.Fatalf("no convention falls back to <mission>-adr.md: %q", got)
	}
}

func TestReserveADRTargetRefusesAMissionThatNeverAccepted(t *testing.T) {
	f := newSQFixture(t, "m1")
	_, err := f.target("m1", domain.StateExecution)
	wantErr(t, err, "adr_side_quest_not_accepted")
}

func TestReserveADRTargetRefusesADeclinedSideQuest(t *testing.T) {
	f := newSQFixture(t, "m1")
	f.mustDecide(t, "m1", SideQuestDeclined, "")
	must(t, SealSideQuestOnGateApproval(f.root, "m1", domain.MissionEventGateApproved, time.Now()))
	_, err := f.target("m1", domain.StateExecution)
	wantErr(t, err, "adr_side_quest_not_accepted")
}

func TestReserveADRTargetRefusesTheWrongStateAndReservesNothing(t *testing.T) {
	f := newSQFixture(t, "m2")
	f.sealed(t, "m2", "")
	_, err := f.target("m2", domain.StateHandoffChallenge)
	wantErr(t, err, "adr_target_wrong_state")
	if rec, _, _ := LoadAcceptedSideQuest(f.root, "m2"); rec.ReservedPath != "" {
		t.Fatal("a refused request reserves nothing")
	}
}

func TestReserveADRTargetNeverOverwritesAnExistingFile(t *testing.T) {
	f := newSQFixture(t, "m1")
	f.sealed(t, "m1", "")
	touch(t, filepath.Join(f.base, "archived"), "m1-adr.md")
	_, err := f.target("m1", domain.StateExecution)
	wantErr(t, err, "adr_target_exists")
}

func TestReserveADRTargetSeparatesMissionsSharingADirectory(t *testing.T) {
	f := newSQFixture(t, "m1")
	touch(t, filepath.Join(f.workspace, "docs", "adr"), "0001-x.md")
	a := newSQFixtureIn(t, f, "ma")
	b := newSQFixtureIn(t, f, "mb")
	a.sealed(t, "ma", "docs/adr")
	b.sealed(t, "mb", "docs/adr")
	if pa, pb := a.mustTarget(t, "ma"), b.mustTarget(t, "mb"); pa == pb {
		t.Fatalf("two missions must not reserve the same number: %q", pa)
	}
}

func newSQFixtureIn(t *testing.T, f sqFixture, missionID string) sqFixture {
	t.Helper()
	pkg := filepath.Join(f.base, "refined", missionID)
	must(t, os.MkdirAll(pkg, 0o750))
	tasks := "```yaml\nside_quests_approved:\n  - {id: OA-ADR-" + missionID + ", status: sq_pending}\n```\n"
	must(t, os.WriteFile(filepath.Join(pkg, "tasks.md"), []byte(tasks), 0o600))
	return f
}

func TestInterruptedMaterializationIsCompletedOnReplay(t *testing.T) {
	f := newSQFixture(t, "m1")
	f.sealed(t, "m1", "")
	rec, _, err := LoadAcceptedSideQuest(f.root, "m1")
	must(t, err)
	// Reservation persisted, claim never recorded (crash in between).
	rec.ReservedPath = ".analysis/archived/m1-adr.md"
	_, err = saveAcceptedSideQuest(f.root, rec)
	must(t, err)
	for range 2 {
		if got := f.mustTarget(t, "m1"); got != rec.ReservedPath {
			t.Fatalf("replay: %q", got)
		}
	}
	if claims := f.claimedPaths(t); len(claims) != 1 {
		t.Fatalf("the missing claim is recorded exactly once: %v", claims)
	}
}
