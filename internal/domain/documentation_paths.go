package domain

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// ValidateProjectRelativeDirectory validates an optional documentation root.
// Empty values are accepted because they preserve legacy fallback behavior.
func ValidateProjectRelativeDirectory(value, field string) error {
	if value == "" {
		return nil
	}
	if err := validateProjectRelative(value); err != nil {
		return fmt.Errorf("%s must be a project-relative directory", field)
	}
	return nil
}

// ValidateProjectRelativePath validates a non-empty workspace-relative file
// path. It shares the same traversal and absolute-path policy as configured
// documentation directories.
func ValidateProjectRelativePath(value, field string) error {
	if value == "" {
		return fmt.Errorf("%s must be a project-relative path", field)
	}
	if err := validateProjectRelative(value); err != nil {
		return fmt.Errorf("%s must be a project-relative path", field)
	}
	return nil
}

func validateProjectRelative(value string) error {
	normalized := strings.ReplaceAll(value, "\\", "/")
	clean := path.Clean(normalized)
	windowsDriveAbsolute := len(normalized) >= 3 && normalized[1] == ':' && normalized[2] == '/'
	if filepath.IsAbs(value) || windowsDriveAbsolute || strings.HasPrefix(normalized, "/") || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("path is not project-relative")
	}
	return nil
}
