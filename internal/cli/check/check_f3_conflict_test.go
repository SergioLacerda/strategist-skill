package check

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func TestParseGitPathLines(t *testing.T) {
	t.Parallel()
	got := parseGitPathLines("\n docs/a.md \n./docs/../docs/b.md\n\n")
	want := []string{"docs/a.md", "docs/b.md"}
	if len(got) != len(want) {
		t.Fatalf("unexpected paths\nwant: %v\n got: %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected paths\nwant: %v\n got: %v", want, got)
		}
	}
}

func TestReadGitConflictedPathsFromWorktree_CommandErrorWrapped(t *testing.T) {
	t.Parallel()
	_, err := readGitConflictedPathsFromWorktree(string([]byte{0}))
	if err == nil {
		t.Fatal("expected command error for invalid worktree")
	}
	if !strings.Contains(err.Error(), "read git conflicted paths") {
		t.Fatalf("expected wrapped context, got %v", err)
	}
}

func TestReadGitConflictedPathsFromWorktree_NonGitDirSkipsGracefully(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // no .git — real non-git worktree, not the invalid-path case above
	paths, err := readGitConflictedPathsFromWorktree(dir)
	if err != nil {
		t.Fatalf("expected graceful skip for non-git worktree, got error: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("expected no conflicted paths for non-git worktree, got: %v", paths)
	}
}

func TestEmitF3ConflictAttributionSignals_EmitsAtThreshold(t *testing.T) {
	// no t.Parallel() — mutates package global and slog default
	root := filepath.Join(t.TempDir(), ".strategist")
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	historyPath := telemetry.SniperMaterializationHistoryPath(root)
	for _, rec := range []telemetry.SniperMaterializationRecord{
		{MissionID: "m-1", BasePath: ".analysis", TargetPath: "docs/a.md", MaterializedAt: now.Add(-time.Hour)},
		{MissionID: "m-2", BasePath: ".analysis", TargetPath: "docs/b.md", MaterializedAt: now.Add(-time.Hour)},
		{MissionID: "m-3", BasePath: ".analysis", TargetPath: "docs/c.md", MaterializedAt: now.Add(-time.Hour)},
	} {
		if err := telemetry.AppendSniperMaterialization(historyPath, rec); err != nil {
			t.Fatalf("append materialization: %v", err)
		}
	}

	origReader := readGitConflictedPaths
	readGitConflictedPaths = func(string) ([]string, error) {
		return []string{"docs/a.md", "docs/b.md", "docs/c.md"}, nil
	}
	t.Cleanup(func() { readGitConflictedPaths = origReader })

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	if err := emitF3ConflictAttributionSignals(root, ".analysis", now); err != nil {
		t.Fatalf("emit f3 signals: %v", err)
	}
	out := buf.String()
	if count := strings.Count(out, "signal=sniper_conflict_attributed"); count != 3 {
		t.Fatalf("expected 3 emitted signals, got %d: %s", count, out)
	}
}

func TestEmitF3ConflictAttributionSignals_ReadGitConflictedPathsError(t *testing.T) {
	// no t.Parallel() — mutates package global readGitConflictedPaths
	origReader := readGitConflictedPaths
	readGitConflictedPaths = func(string) ([]string, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { readGitConflictedPaths = origReader })

	root := filepath.Join(t.TempDir(), ".strategist")
	err := emitF3ConflictAttributionSignals(root, ".analysis", time.Now())
	if err == nil {
		t.Fatal("expected error to propagate from readGitConflictedPaths")
	}
	if err.Error() != "boom" {
		t.Fatalf("expected unwrapped propagation, got: %v", err)
	}
}

func TestEmitF3ConflictAttributionSignals_ReadRecentMaterializationsError(t *testing.T) {
	// no t.Parallel() — mutates package global readGitConflictedPaths
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	origReader := readGitConflictedPaths
	readGitConflictedPaths = func(string) ([]string, error) { return nil, nil }
	t.Cleanup(func() { readGitConflictedPaths = origReader })

	root := filepath.Join(t.TempDir(), ".strategist")
	historyPath := telemetry.SniperMaterializationHistoryPath(root)
	if err := os.MkdirAll(filepath.Dir(historyPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(historyPath, []byte(`{"mission_id":"m"}`), 0o000); err != nil {
		t.Fatalf("write: %v", err)
	}
	if os.Getuid() == 0 {
		t.Skip("running as root — file permission checks do not apply")
	}

	err := emitF3ConflictAttributionSignals(root, ".analysis", time.Now())
	if err == nil {
		t.Fatal("expected error for unreadable materializations history file")
	}
	if !strings.Contains(err.Error(), "read recent sniper materializations") {
		t.Fatalf("expected wrapped context, got: %v", err)
	}
}

func TestEmitF3ConflictAttributionSignals_EmitsClaimCollisionSignal(t *testing.T) {
	// no t.Parallel() — mutates package global and slog default
	root := filepath.Join(t.TempDir(), ".strategist")
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	claimPath := telemetry.SniperClaimHistoryPath(root)
	for _, rec := range []telemetry.SniperClaimRecord{
		{MissionID: "m-1", BasePath: ".analysis", TargetPath: "docs/a.md", ClaimedAt: now.Add(-time.Hour)},
		{MissionID: "m-2", BasePath: ".analysis", TargetPath: "docs/a.md", ClaimedAt: now.Add(-time.Minute)},
	} {
		if err := telemetry.AppendSniperClaim(claimPath, rec); err != nil {
			t.Fatalf("append claim: %v", err)
		}
	}

	origReader := readGitConflictedPaths
	readGitConflictedPaths = func(string) ([]string, error) { return nil, nil }
	t.Cleanup(func() { readGitConflictedPaths = origReader })

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	if err := emitF3ConflictAttributionSignals(root, ".analysis", now); err != nil {
		t.Fatalf("emit f3 signals: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "signal=sniper_claim_collision") {
		t.Fatalf("expected claim collision signal to be emitted, got: %s", out)
	}
	if !strings.Contains(out, "docs/a.md") {
		t.Fatalf("expected target path in emitted signal, got: %s", out)
	}
}

func TestEmitF3ConflictAttributionSignals_ReadRecentClaimsError(t *testing.T) {
	// no t.Parallel() — mutates package global readGitConflictedPaths
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	origReader := readGitConflictedPaths
	readGitConflictedPaths = func(string) ([]string, error) { return nil, nil }
	t.Cleanup(func() { readGitConflictedPaths = origReader })

	root := filepath.Join(t.TempDir(), ".strategist")
	claimPath := telemetry.SniperClaimHistoryPath(root)
	if err := os.MkdirAll(filepath.Dir(claimPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(claimPath, []byte(`{"mission_id":"m"}`), 0o000); err != nil {
		t.Fatalf("write: %v", err)
	}
	if os.Getuid() == 0 {
		t.Skip("running as root — file permission checks do not apply")
	}

	err := emitF3ConflictAttributionSignals(root, ".analysis", time.Now())
	if err == nil {
		t.Fatal("expected error for unreadable claim history file")
	}
	if !strings.Contains(err.Error(), "read recent sniper claims") {
		t.Fatalf("expected wrapped context, got: %v", err)
	}
}

// writeMissionPhase writes a minimal missions/<id>.json with just enough
// shape for ReadMissionPhase's json.Unmarshal into domain.MissionEngineStatus
// to succeed. It deliberately does not go through domain.RestoreMission's
// full validation — ReadMissionPhase doesn't either, by design (see its doc
// comment).
func writeMissionPhase(t *testing.T, root, missionID, phase string) {
	t.Helper()
	dir := filepath.Join(root, "missions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir missions: %v", err)
	}
	body := `{"mission_id":"` + missionID + `","phase":"` + phase + `","state":"IGNORED"}`
	if err := os.WriteFile(filepath.Join(dir, missionID+".json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write mission state: %v", err)
	}
}

// TestEmitF3ConflictAttributionSignals_TerminalMissionClaimReleased covers
// ADR-0057 § A2 / design.md task 2.4: a claim from a mission that has already
// reached DONE is excluded from collision detection, so two sequential,
// fully completed missions touching the same target inside
// SniperClaimWindow do not report a false collision.
func TestEmitF3ConflictAttributionSignals_TerminalMissionClaimReleased(t *testing.T) {
	// no t.Parallel() — mutates package global and slog default
	root := filepath.Join(t.TempDir(), ".strategist")
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	claimPath := telemetry.SniperClaimHistoryPath(root)
	for _, rec := range []telemetry.SniperClaimRecord{
		{MissionID: "m-done-1", BasePath: ".analysis", TargetPath: "docs/a.md", ClaimedAt: now.Add(-48 * time.Hour)},
		{MissionID: "m-done-2", BasePath: ".analysis", TargetPath: "docs/a.md", ClaimedAt: now.Add(-time.Hour)},
	} {
		if err := telemetry.AppendSniperClaim(claimPath, rec); err != nil {
			t.Fatalf("append claim: %v", err)
		}
	}
	// Both missions completed — their claims on the same target must not
	// collide.
	writeMissionPhase(t, root, "m-done-1", "DONE")
	writeMissionPhase(t, root, "m-done-2", "DONE")

	origReader := readGitConflictedPaths
	readGitConflictedPaths = func(string) ([]string, error) { return nil, nil }
	t.Cleanup(func() { readGitConflictedPaths = origReader })

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	if err := emitF3ConflictAttributionSignals(root, ".analysis", now); err != nil {
		t.Fatalf("emit f3 signals: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "signal=sniper_claim_collision") {
		t.Fatalf("expected no claim collision signal once both missions are terminal, got: %s", out)
	}
}

// TestEmitF3ConflictAttributionSignals_InFlightMissionClaimsStillCollide is
// the sibling of the terminal-release test: two missions that have NOT
// reached a terminal phase still collide on the same target, and a mission
// with no readable state file at all is conservatively treated as still
// live rather than silently excused.
func TestEmitF3ConflictAttributionSignals_InFlightMissionClaimsStillCollide(t *testing.T) {
	// no t.Parallel() — mutates package global and slog default
	root := filepath.Join(t.TempDir(), ".strategist")
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	claimPath := telemetry.SniperClaimHistoryPath(root)
	for _, rec := range []telemetry.SniperClaimRecord{
		{MissionID: "m-inflight", BasePath: ".analysis", TargetPath: "docs/b.md", ClaimedAt: now.Add(-time.Hour)},
		{MissionID: "m-unreadable", BasePath: ".analysis", TargetPath: "docs/b.md", ClaimedAt: now.Add(-time.Minute)},
	} {
		if err := telemetry.AppendSniperClaim(claimPath, rec); err != nil {
			t.Fatalf("append claim: %v", err)
		}
	}
	// m-inflight has an explicit non-terminal phase; m-unreadable has no
	// state file at all (found=false), which must still be treated as live.
	writeMissionPhase(t, root, "m-inflight", "EXECUTION")

	origReader := readGitConflictedPaths
	readGitConflictedPaths = func(string) ([]string, error) { return nil, nil }
	t.Cleanup(func() { readGitConflictedPaths = origReader })

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	if err := emitF3ConflictAttributionSignals(root, ".analysis", now); err != nil {
		t.Fatalf("emit f3 signals: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "signal=sniper_claim_collision") {
		t.Fatalf("expected claim collision signal for two in-flight/unreadable missions, got: %s", out)
	}
	if !strings.Contains(out, "docs/b.md") {
		t.Fatalf("expected target path in emitted signal, got: %s", out)
	}
}

func TestReadGitConflictedPathsFromWorktree_CleanRepoSuccess(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	initGitRepo(t, dir)

	paths, err := readGitConflictedPathsFromWorktree(dir)
	if err != nil {
		t.Fatalf("expected no error for clean repo, got: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("expected no conflicted paths in clean repo, got: %v", paths)
	}
}

func TestReadGitConflictedPathsFromWorktree_DiffCommandFails(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	initGitRepo(t, dir)
	// Corrupt the index so `git rev-parse` (which doesn't read it) still
	// succeeds, but `git diff` (which does) fails — isolating the second
	// command's error branch from the "not a git repo" early return.
	if err := os.WriteFile(filepath.Join(dir, ".git", "index"), []byte("not-an-index"), 0o644); err != nil {
		t.Fatalf("corrupt index: %v", err)
	}

	_, err := readGitConflictedPathsFromWorktree(dir)
	if err == nil {
		t.Fatal("expected error from corrupted git index")
	}
	if !strings.Contains(err.Error(), "read git conflicted paths") {
		t.Fatalf("expected wrapped context, got: %v", err)
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", ".")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run("add", "a.txt")
	run("commit", "-q", "-m", "init")
}
