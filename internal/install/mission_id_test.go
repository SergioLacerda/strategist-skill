package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)

func TestGenerateMissionID_NoCollision(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	id := GenerateMissionID(dir, "my feature", fixedNow)
	if id != "20260602-my-feature" {
		t.Fatalf("unexpected id: %s", id)
	}
}

func TestGenerateMissionID_SlugSanitized(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cases := []struct{ slug, want string }{
		{"My Feature", "20260602-my-feature"},
		{"hello_world", "20260602-hello-world"},
		{"YAML Round-Trip", "20260602-yaml-round-trip"},
		{"  spaces  ", "20260602-spaces"},
	}
	for _, tc := range cases {
		got := GenerateMissionID(dir, tc.slug, fixedNow)
		if got != tc.want {
			t.Fatalf("slug=%q: want %s, got %s", tc.slug, tc.want, got)
		}
	}
}

func TestGenerateMissionID_CollisionAddsuffix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create a file that collides.
	if err := os.MkdirAll(filepath.Join(dir, "pending"), 0o755); err != nil {
		t.Fatal(err)
	}
	collidingFile := filepath.Join(dir, "pending", "20260602-my-feature-discovery.md")
	if err := os.WriteFile(collidingFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	id := GenerateMissionID(dir, "my feature", fixedNow)
	if id == "20260602-my-feature" {
		t.Fatal("expected suffix on collision, got base id")
	}
	if !strings.HasPrefix(id, "20260602-my-feature-") {
		t.Fatalf("expected prefix 20260602-my-feature-, got %s", id)
	}
}

func TestPatternCollides_MalformedPatternReturnsFalse(t *testing.T) {
	t.Parallel()
	// An unterminated "[" makes filepath.Glob return ErrBadPattern.
	if patternCollides("[") {
		t.Fatal("expected malformed pattern to be treated as no collision")
	}
}
