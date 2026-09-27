package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareResolvedDigest_Match(t *testing.T) {
	t.Parallel()
	path := writeCatalogFixture(t, t.TempDir(), "  - id: brainstorming\n    risk_score: write_analysis\n    upstream_content_digest: sha256:abc\n")
	result, err := CompareResolvedDigest(path, "brainstorming", "sha256:abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != ResolvedDigestMatch {
		t.Fatalf("status = %v, want match", result.Status)
	}
}

func TestCompareResolvedDigest_Mismatch(t *testing.T) {
	t.Parallel()
	path := writeCatalogFixture(t, t.TempDir(), "  - id: brainstorming\n    risk_score: write_analysis\n    upstream_content_digest: sha256:abc\n")
	result, err := CompareResolvedDigest(path, "brainstorming", "sha256:def")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != ResolvedDigestMismatch {
		t.Fatalf("status = %v, want mismatch", result.Status)
	}
	if result.Pin != "sha256:abc" || result.Resolved != "sha256:def" {
		t.Fatalf("result = %+v", result)
	}
}

func TestCompareResolvedDigest_UnavailablePinIsUnknownNeverMatch(t *testing.T) {
	t.Parallel()
	path := writeCatalogFixture(t, t.TempDir(), "  - id: sniper\n    risk_score: controlled\n")
	result, err := CompareResolvedDigest(path, "sniper", "sha256:anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != ResolvedDigestPinUnavailable {
		t.Fatalf("status = %v, want pin-unavailable (never a match)", result.Status)
	}
}

func TestCompareResolvedDigest_EmptyResolvedIsUnknown(t *testing.T) {
	t.Parallel()
	path := writeCatalogFixture(t, t.TempDir(), "  - id: brainstorming\n    risk_score: write_analysis\n    upstream_content_digest: sha256:abc\n")
	result, err := CompareResolvedDigest(path, "brainstorming", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != ResolvedDigestNotReported {
		t.Fatalf("status = %v, want not-reported", result.Status)
	}
}

func TestHashFileSHA256_MatchesKnownContent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := HashFileSHA256(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
