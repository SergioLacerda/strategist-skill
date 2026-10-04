package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SlotWriteScope declares the write boundary for a slot provider.
type SlotWriteScope struct {
	SlotName      string // e.g. "discovery", "refinement", "execution"
	AllowedPrefix string // e.g. ".analysis/pending/"
	AllowedExt    string // e.g. ".md" — empty means any extension is allowed
}

// requiredArchivistFiles lists the four files Archivist must produce in the refined package.
var requiredArchivistFiles = []string{"analysis.md", "proposal.md", "design.md", "tasks.md"}

// FilePresence is the filesystem port used by package completeness checks.
// Concrete workspace access belongs to an adapter or caller.
type FilePresence interface {
	Exists(path string) bool
}

// ValidateArchivistPackage returns an error if the refined package directory is missing
// any of the four required files. This enforces the four-file completeness invariant:
// a package with analysis.md absent is incomplete even if the other three files exist.
func ValidateArchivistPackage(refinedDir string, presence FilePresence) error {
	var missing []string
	for _, f := range requiredArchivistFiles {
		path := filepath.Join(refinedDir, f)
		if !presence.Exists(path) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("archivist_package_incomplete: missing files in %s: %v", refinedDir, missing)
	}
	return nil
}

// ValidateSlotWrite returns an error if attemptedPath violates the declared scope.
// The error message always contains "slot_write_scope_violation" so callers and
// BDD scenarios can match on that token.
func ValidateSlotWrite(scope SlotWriteScope, attemptedPath string) error {
	if !PathWithin(scope.AllowedPrefix, attemptedPath) {
		return fmt.Errorf("slot_write_scope_violation: %s attempted write to %q (allowed prefix: %q)",
			scope.SlotName, attemptedPath, scope.AllowedPrefix)
	}
	if scope.AllowedExt != "" && filepath.Ext(attemptedPath) != scope.AllowedExt {
		return fmt.Errorf("slot_write_scope_violation: %s non-%s type at %q",
			scope.SlotName, scope.AllowedExt, attemptedPath)
	}
	return nil
}

// PathWithin reports whether attemptedPath is the same path as allowedPrefix
// or a descendant of it, using the host filesystem's path semantics. It is a
// pure boundary primitive shared by domain validators and executable Feats.
func PathWithin(allowedPrefix, attemptedPath string) bool {
	if attemptedPath == "" || allowedPrefix == "" {
		return false
	}

	allowed := filepath.Clean(allowedPrefix)
	attempted := filepath.Clean(attemptedPath)
	if filepath.IsAbs(allowed) != filepath.IsAbs(attempted) {
		return false
	}

	rel, err := filepath.Rel(allowed, attempted)
	if err != nil {
		return false
	}
	return rel == "." || (!filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
