package domain

import (
	"fmt"
	"strings"
)

// ValidateTaxonomyVersion accepts an empty value for artifacts written before
// taxonomy versioning was introduced, but rejects explicit unknown versions.
func ValidateTaxonomyVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" || version == CanonicalTaxonomyVersion {
		return nil
	}
	return fmt.Errorf("unsupported taxonomy version %q", version)
}
