package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// identityRelPaths are the internal-domain identity files that templates/domain/index.yaml
// lists under load_always. Preflight treats their absence as a soft, easy-to-miss warn
// (identity=degraded); this check makes the same condition loud and non-zero at the CLI
// level (D6) without changing preflight's own backward-compatible degraded emission.
var identityRelPaths = []string{
	filepath.Join("templates", "domain", "identity", "drift-patterns.yaml"),
	filepath.Join("templates", "domain", "identity", "what-i-am.yaml"),
}

// identityFilesMissing returns the identity file paths (relative to root, forward-slash
// form for display) that do not exist under root.
func identityFilesMissing(root string) []string {
	var missing []string
	for _, rel := range identityRelPaths {
		if _, err := os.Stat(filepath.Join(root, rel)); os.IsNotExist(err) {
			missing = append(missing, filepath.ToSlash(rel))
		}
	}
	return missing
}

// checkIdentityFilesBlockingError returns the check=blocked error for missing identity
// files, or nil when both are present.
func checkIdentityFilesBlockingError(root string) error {
	missing := identityFilesMissing(root)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"[Strategist] check=blocked reason=identity_files_missing files_missing=%s\n→ Run: strategist compile",
		strings.Join(missing, ","),
	)
}
