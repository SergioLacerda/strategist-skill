package domain

import (
	"path"
	"path/filepath"
	"strings"
)

// isSafeRuntimeRoot accepts only a canonical slash-separated path under
// .strategist. The separator is normalized before the cleanliness comparison:
// on Windows filepath.Clean rewrites "/" to "\\", so comparing the raw string
// against its cleaned form would reject the catalog's own declaration.
func isSafeRuntimeRoot(root string) bool {
	if root == "" || filepath.IsAbs(root) {
		return false
	}
	slash := filepath.ToSlash(root)
	if !hasSafeRuntimePrefix(slash) {
		return false
	}
	return safeRuntimeSegments(slash)
}

func hasSafeRuntimePrefix(slash string) bool {
	return !path.IsAbs(slash) && path.Clean(slash) == slash && strings.HasPrefix(slash, ".strategist/")
}

func safeRuntimeSegments(slash string) bool {
	for _, segment := range strings.Split(slash, "/") {
		if segment == ".." || strings.Contains(segment, ":") {
			return false
		}
	}
	return true
}
