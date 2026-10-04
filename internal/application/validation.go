// Package application owns consumer-facing orchestration services. Adapters
// such as Cobra and filesystem entrypoints call these services; domain
// packages remain independent of them.
package application

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/validate"
)

// ValidationReport is the read-only result of validating an installed
// Strategist runtime.
type ValidationReport struct {
	Checks int
	Errors []string
}

// ValidateWorkspace validates the configuration tree rooted at a discovered
// .strategist directory. It performs no writes and leaves rendering to the
// caller.
func ValidateWorkspace(root string) ValidationReport {
	report := ValidationReport{}

	activeErr := validate.ActiveYAML(filepath.Join(root, "active.yaml"))
	report.Checks++
	if activeErr != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("active.yaml: %v", activeErr))
	}

	personaErrs, personaChecks := validate.PersonasDir(filepath.Join(root, "personas"))
	report.Checks += personaChecks
	report.Errors = append(report.Errors, personaErrs...)

	roleErrs, roleChecks := validate.RolesDir(filepath.Join(root, "roles"))
	report.Checks += roleChecks
	report.Errors = append(report.Errors, roleErrs...)

	knowledgeIndex := filepath.Join(root, "knowledge.index.yaml")
	if fileExists(knowledgeIndex) {
		report.Checks++
		if err := validate.YAMLFile(knowledgeIndex); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("knowledge.index.yaml: %v", err))
		}
	}
	return report
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
