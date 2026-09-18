package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestMaterializeContext_IsOrderedBoundedAndVerifiesDigest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.md"), []byte("bravo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("alpha"))
	result, err := MaterializeContext(root, []ContextReference{
		{Ref: "b.md", Kind: "context"},
		{Ref: "a.md", Kind: "context", Digest: "sha256:" + hex.EncodeToString(sum[:])},
	}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.References) != 2 || result.References[0].Ref != "a.md" || result.References[1].Ref != "b.md" {
		t.Fatalf("unexpected order: %+v", result.References)
	}
	if result.Digest == "" || result.References[0].Provenance == "" {
		t.Fatalf("missing identity: %+v", result)
	}
}

func TestMaterializeContext_BlocksMissingDigestDuplicateAndBounds(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name              string
		refs              []ContextReference
		maxRefs, maxBytes int
	}{
		{"missing", []ContextReference{{Ref: "missing.md", Kind: "context"}}, 2, 20},
		{"digest", []ContextReference{{Ref: "a.md", Kind: "context", Digest: "sha256:bad"}}, 2, 20},
		{"duplicate", []ContextReference{{Ref: "a.md", Kind: "context"}, {Ref: "a.md", Kind: "context"}}, 3, 20},
		{"bytes bound", []ContextReference{{Ref: "a.md", Kind: "context"}}, 2, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := MaterializeContext(root, tc.refs, tc.maxRefs, tc.maxBytes); err == nil {
				t.Fatal("expected blocked materialization")
			}
		})
	}
}
