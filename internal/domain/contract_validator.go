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

// ValidateSlotWrite returns an error if attemptedPath violates the declared scope.
// The error message always contains "slot_write_scope_violation" so callers and
// BDD scenarios can match on that token.
func ValidateSlotWrite(scope SlotWriteScope, attemptedPath string) error {
	if !strings.HasPrefix(attemptedPath, scope.AllowedPrefix) {
		return fmt.Errorf("slot_write_scope_violation: %s attempted write to %q (allowed prefix: %q)",
			scope.SlotName, attemptedPath, scope.AllowedPrefix)
	}
	if scope.AllowedExt != "" && filepath.Ext(attemptedPath) != scope.AllowedExt {
		return fmt.Errorf("slot_write_scope_violation: %s non-%s type at %q",
			scope.SlotName, scope.AllowedExt, attemptedPath)
	}
	return nil
}
