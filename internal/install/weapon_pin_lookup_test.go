package install

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCatalogFixture(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "catalog.yaml")
	if err := os.WriteFile(path, []byte("schema_version: "+pluginCatalogSchemaVersion+"\nproviders:\n"+body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestReadUpstreamContentDigest_Found(t *testing.T) {
	t.Parallel()
	path := writeCatalogFixture(t, t.TempDir(), "  - id: brainstorming\n    risk_score: write_analysis\n    upstream_content_digest: sha256:abc\n")
	digest, found, err := ReadUpstreamContentDigest(path, "brainstorming")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || digest != "sha256:abc" {
		t.Fatalf("found=%v digest=%q", found, digest)
	}
}

func TestReadUpstreamContentDigest_ProviderWithNoPin(t *testing.T) {
	t.Parallel()
	path := writeCatalogFixture(t, t.TempDir(), "  - id: sniper\n    risk_score: controlled\n")
	digest, found, err := ReadUpstreamContentDigest(path, "sniper")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found || digest != "" {
		t.Fatalf("a provider with no upstream_content_digest must report found=false, got found=%v digest=%q", found, digest)
	}
}

func TestReadUpstreamContentDigest_UnknownProvider(t *testing.T) {
	t.Parallel()
	path := writeCatalogFixture(t, t.TempDir(), "  - id: brainstorming\n    risk_score: write_analysis\n    upstream_content_digest: sha256:abc\n")
	_, _, err := ReadUpstreamContentDigest(path, "no-such-weapon")
	if err == nil {
		t.Fatal("want an error for an unknown provider id")
	}
}

func TestReadUpstreamContentDigest_MissingCatalog(t *testing.T) {
	t.Parallel()
	_, _, err := ReadUpstreamContentDigest(filepath.Join(t.TempDir(), "absent.yaml"), "brainstorming")
	if err == nil {
		t.Fatal("want an error for a missing catalog file")
	}
}
