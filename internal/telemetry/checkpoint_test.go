package telemetry

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCheckpoint_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "mission-test-checkpoint.json")

	cp := &MissionCheckpoint{
		MissionID:   "m-round",
		TasksTotal:  5,
		LastUpdated: time.Now().Truncate(time.Second),
	}
	cp.MarkTaskDone(1)
	cp.MarkTaskDone(2)

	if err := SaveCheckpoint(path, cp); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadCheckpoint(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.MissionID != cp.MissionID {
		t.Fatalf("mission id mismatch: %s != %s", loaded.MissionID, cp.MissionID)
	}
	if !loaded.TaskDone(1) || !loaded.TaskDone(2) {
		t.Fatalf("completed tasks not persisted: %v", loaded.TasksCompleted)
	}
	if loaded.TaskDone(3) {
		t.Fatal("task 3 should not be done")
	}
}

func TestLoadCheckpoint_MissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cp, err := LoadCheckpoint(filepath.Join(dir, "nonexistent.json"))
	if err != nil {
		t.Fatalf("missing file should return empty checkpoint, got error: %v", err)
	}
	if cp == nil {
		t.Fatal("expected non-nil checkpoint")
	}
	if len(cp.TasksCompleted) != 0 {
		t.Fatalf("expected empty completed list, got %v", cp.TasksCompleted)
	}
}

func TestCheckpoint_MarkTaskDone_Idempotent(t *testing.T) {
	t.Parallel()
	cp := &MissionCheckpoint{MissionID: "m-idem", TasksTotal: 3}
	cp.MarkTaskDone(1)
	cp.MarkTaskDone(1)
	cp.MarkTaskDone(1)
	if len(cp.TasksCompleted) != 1 {
		t.Fatalf("expected 1 entry, got %d: %v", len(cp.TasksCompleted), cp.TasksCompleted)
	}
}

func TestRemoveCheckpoint(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "rm-test.json")

	cp := &MissionCheckpoint{MissionID: "m-rm", TasksTotal: 1}
	if err := SaveCheckpoint(path, cp); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := RemoveCheckpoint(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	// Second remove of missing file must not error.
	if err := RemoveCheckpoint(path); err != nil {
		t.Fatalf("double remove: %v", err)
	}
}

func TestLoadCheckpoint_CorruptJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadCheckpoint(path)
	if err == nil {
		t.Fatal("expected error for corrupt JSON, got nil")
	}
}

func TestLoadCheckpoint_UnreadableFile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "locked.json")
	if err := os.WriteFile(path, []byte(`{"mission_id":"m"}`), 0o000); err != nil {
		t.Fatalf("write: %v", err)
	}
	if os.Getuid() == 0 {
		t.Skip("running as root — file permission checks do not apply")
	}
	_, err := LoadCheckpoint(path)
	if err == nil {
		t.Fatal("expected error for unreadable file, got nil")
	}
}

func TestSaveCheckpoint_UnwritableDir(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if os.Getuid() == 0 {
		t.Skip("running as root — dir permission checks do not apply")
	}
	cp := &MissionCheckpoint{MissionID: "m-fail", TasksTotal: 1}
	err := SaveCheckpoint(filepath.Join(dir, "cp.json"), cp)
	if err == nil {
		t.Fatal("expected error for unwritable dir, got nil")
	}
}

func TestSaveCheckpoint_RenameFailsWhenPathIsDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// path itself is an existing directory — os.Rename(tmp, path) fails.
	target := filepath.Join(dir, "cp-dir")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cp := &MissionCheckpoint{MissionID: "m-rename-fail", TasksTotal: 1}
	err := SaveCheckpoint(target, cp)
	if err == nil {
		t.Fatal("expected error when destination path is a directory, got nil")
	}

	// The temp file must be cleaned up, not left behind in the parent dir.
	matches, globErr := filepath.Glob(filepath.Join(dir, ".cp-dir-*.tmp"))
	if globErr != nil {
		t.Fatalf("glob: %v", globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("expected temp file cleanup, found: %v", matches)
	}
}

func TestRemoveCheckpoint_UnremovableFile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root — dir permission checks do not apply")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "locked.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Make parent dir read-only so os.Remove fails.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	err := RemoveCheckpoint(path)
	if err == nil {
		t.Fatal("expected error for unremovable file, got nil")
	}
}

func TestCheckpoint_SkipCompletedTasks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "idempotency.json")

	// Simulate Sniper completing tasks 1-3 before failure.
	cp := &MissionCheckpoint{MissionID: "m-skip", TasksTotal: 5}
	for _, task := range []int{1, 2, 3} {
		cp.MarkTaskDone(task)
	}
	if err := SaveCheckpoint(path, cp); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadCheckpoint(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	assertSkippedTasks(t, loaded, 3, 5)
	assertTasksNotDone(t, loaded, 4, 5)
}

func assertSkippedTasks(t *testing.T, cp *MissionCheckpoint, want, total int) {
	t.Helper()
	skipped := 0
	for task := 1; task <= total; task++ {
		if cp.TaskDone(task) {
			skipped++
		}
	}
	if skipped != want {
		t.Fatalf("expected %d tasks skipped, got %d", want, skipped)
	}
}

func assertTasksNotDone(t *testing.T, cp *MissionCheckpoint, tasks ...int) {
	t.Helper()
	for _, task := range tasks {
		if cp.TaskDone(task) {
			t.Fatalf("task %d should not be done", task)
		}
	}
}
