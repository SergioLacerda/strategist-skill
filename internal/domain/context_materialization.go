package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"
)

const (
	// DefaultContextMaxReferences bounds the number of structural references.
	DefaultContextMaxReferences = 64
	// DefaultContextMaxBytes bounds the total materialized context size.
	DefaultContextMaxBytes = 1 << 20
)

// ContextReference identifies one workspace-relative structural input.
type ContextReference struct {
	Ref    string `json:"ref"`
	Kind   string `json:"kind"`
	Digest string `json:"digest,omitempty"`
}

// ContextReader is the filesystem port used by context materialization. The
// domain validates references and computes deterministic evidence; callers own
// how a reference is read from a workspace, archive, or test fixture.
type ContextReader interface {
	ReadFile(ref string) ([]byte, error)
}

// MaterializedContext is the verified, deterministic context result.
type MaterializedContext struct {
	References []MaterializedReference `json:"references"`
	Bytes      int                     `json:"bytes"`
	Digest     string                  `json:"digest"`
}

// MaterializedReference contains content and its integrity/provenance data.
type MaterializedReference struct {
	Ref        string `json:"ref"`
	Kind       string `json:"kind"`
	Digest     string `json:"digest"`
	Provenance string `json:"provenance"`
	Content    string `json:"content"`
}

// MaterializeContext resolves only declared references through reader. It is
// deterministic, bounded, and deliberately does not synthesize content.
func MaterializeContext(reader ContextReader, refs []ContextReference, maxRefs, maxBytes int) (MaterializedContext, error) {
	maxBytes, err := validateMaterializationRequest(reader, refs, maxRefs, maxBytes)
	if err != nil {
		return MaterializedContext{}, err
	}
	ordered := sortContextReferences(refs)
	result := MaterializedContext{References: make([]MaterializedReference, 0, len(ordered))}
	seen := make(map[string]struct{}, len(ordered))
	for _, ref := range ordered {
		materialized, err := materializeReference(reader, ref, seen, maxBytes-result.Bytes)
		if err != nil {
			return MaterializedContext{}, err
		}
		seen[ref.Ref+"\x00"+ref.Kind] = struct{}{}
		result.Bytes += len(materialized.Content)
		result.References = append(result.References, materialized)
	}
	result.Digest = materializedContextDigest(result.References)
	return result, nil
}

func materializeReference(reader ContextReader, ref ContextReference, seen map[string]struct{}, remaining int) (MaterializedReference, error) {
	_, err := validateContextReference(ref, seen)
	if err != nil {
		return MaterializedReference{}, err
	}
	data, err := reader.ReadFile(ref.Ref)
	if err != nil {
		return MaterializedReference{}, fmt.Errorf("context materialization: blocked reference %q: %w", ref.Ref, err)
	}
	if len(data) > remaining {
		return MaterializedReference{}, fmt.Errorf("context materialization: byte limit exceeded")
	}
	digest := contextDigest(data)
	if ref.Digest != "" && ref.Digest != digest {
		return MaterializedReference{}, fmt.Errorf("context materialization: blocked digest mismatch for %q", ref.Ref)
	}
	return MaterializedReference{Ref: ref.Ref, Kind: ref.Kind, Digest: digest, Provenance: "workspace:" + ref.Ref, Content: string(data)}, nil
}

func validateMaterializationRequest(reader ContextReader, refs []ContextReference, maxRefs, maxBytes int) (int, error) {
	if reader == nil {
		return 0, fmt.Errorf("context materialization: reader is required")
	}
	if maxRefs <= 0 {
		maxRefs = DefaultContextMaxReferences
	}
	if maxBytes <= 0 {
		maxBytes = DefaultContextMaxBytes
	}
	if len(refs) > maxRefs {
		return 0, fmt.Errorf("context materialization: reference limit exceeded (%d > %d)", len(refs), maxRefs)
	}
	return maxBytes, nil
}

func sortContextReferences(refs []ContextReference) []ContextReference {
	ordered := append([]ContextReference(nil), refs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Ref != ordered[j].Ref {
			return ordered[i].Ref < ordered[j].Ref
		}
		return ordered[i].Kind < ordered[j].Kind
	})
	return ordered
}

func validateContextReference(ref ContextReference, seen map[string]struct{}) (string, error) {
	if ref.Ref == "" || path.IsAbs(ref.Ref) || path.Clean(ref.Ref) != ref.Ref || ref.Ref == "." || ref.Ref == ".." || containsParentPathComponent(ref.Ref) {
		return "", fmt.Errorf("context materialization: invalid relative reference %q", ref.Ref)
	}
	key := ref.Ref + "\x00" + ref.Kind
	if _, ok := seen[key]; ok {
		return "", fmt.Errorf("context materialization: duplicate reference %q", ref.Ref)
	}
	return key, nil
}

func containsParentPathComponent(ref string) bool {
	for _, component := range strings.Split(strings.ReplaceAll(ref, `\`, "/"), "/") {
		if component == ".." {
			return true
		}
	}
	return false
}

func contextDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func materializedContextDigest(refs []MaterializedReference) string {
	hash := sha256.New()
	for _, ref := range refs {
		if _, err := fmt.Fprintf(hash, "%s\x00%s\x00%s\x00", ref.Ref, ref.Kind, ref.Digest); err != nil {
			return ""
		}
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}
