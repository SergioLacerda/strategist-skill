package refinement

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeComponent = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func validateInput(input OpenSpecInput) error {
	if err := validateComponents(input); err != nil {
		return err
	}
	if err := validateAbsolutePaths(input); err != nil {
		return err
	}
	if err := validateContained(input.BasePath, input.PendingAnalysisPath); err != nil {
		return fmt.Errorf("pending analysis path: %w", err)
	}
	if filepath.Clean(input.RuntimeRoot) == filepath.Clean(input.BasePath) {
		return fmt.Errorf("runtime root must remain separate from base path")
	}
	return nil
}

func validateComponents(input OpenSpecInput) error {
	for label, value := range map[string]string{"mission id": input.MissionID, "change id": input.ChangeID} {
		if !safeComponent.MatchString(value) {
			return fmt.Errorf("%s is malformed", label)
		}
	}
	return nil
}

func validateAbsolutePaths(input OpenSpecInput) error {
	for label, value := range map[string]string{"base path": input.BasePath, "runtime root": input.RuntimeRoot, "pending analysis path": input.PendingAnalysisPath} {
		if value == "" || !filepath.IsAbs(value) {
			return fmt.Errorf("%s must be an absolute path", label)
		}
	}
	return nil
}

func validatedChangeDir(input OpenSpecInput) (string, error) {
	changeDir := filepath.Join(input.RuntimeRoot, "changes", input.ChangeID)
	if err := validateContained(input.RuntimeRoot, changeDir); err != nil {
		return "", fmt.Errorf("openspec bridge: change path: %w", err)
	}
	return changeDir, nil
}

func validatedRefinedPath(input OpenSpecInput) (string, error) {
	refined := filepath.Join(input.BasePath, "refined", input.MissionID)
	if err := validateContained(input.BasePath, refined); err != nil {
		return "", fmt.Errorf("openspec bridge: refined path: %w", err)
	}
	return refined, nil
}

func validateContained(root, target string) error {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes root")
	}
	return nil
}

func requireFiles(root string, names []string) error {
	for _, name := range names {
		info, err := os.Stat(filepath.Join(root, name))
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", name)
		}
	}
	return nil
}
