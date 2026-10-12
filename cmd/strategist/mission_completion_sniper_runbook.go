package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runbook"
)

// requireValidRunbookSidecars verifies, at Sniper completion, every typed
// runbook sidecar of the snapshot with the same parser the selector uses, and
// checks that its identity agrees with what the resolver declared. A runbook
// that would later be silently skipped by the selector is therefore refused
// while the mission can still repair it.
func requireValidRunbookSidecars(workspaceRoot string, snapshot domain.ExecutionSnapshot) error {
	for _, artifact := range runbookSidecarArtifacts(snapshot) {
		if err := verifyRunbookSidecar(workspaceRoot, artifact); err != nil {
			return err
		}
	}
	return nil
}

func runbookSidecarArtifacts(snapshot domain.ExecutionSnapshot) []domain.ExecutionArtifact {
	var sidecars []domain.ExecutionArtifact
	for _, group := range snapshot.Groups {
		for _, artifact := range group.Artifacts {
			if artifact.Kind == domain.RunbookSidecarKind {
				sidecars = append(sidecars, artifact)
			}
		}
	}
	return sidecars
}

func verifyRunbookSidecar(workspaceRoot string, artifact domain.ExecutionArtifact) error {
	raw, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(artifact.Path))) //nolint:gosec // path is an authorized snapshot artifact
	if err != nil {
		return fmt.Errorf("role_invocation_failed: runbook sidecar %q is unreadable: %w", artifact.Path, err)
	}
	rb, err := runbook.ParseSidecar(raw)
	if err != nil {
		return fmt.Errorf("role_invocation_failed: runbook sidecar %q: %w", artifact.Path, err)
	}
	if want := artifact.Metadata["runbook_id"]; want != "" && rb.RunbookID != want {
		return fmt.Errorf("role_invocation_failed: runbook sidecar %q declares runbook_id %q, expected %q", artifact.Path, rb.RunbookID, want)
	}
	for _, finding := range runbook.CheckSidecarConsistency(artifact.Path, rb, workspaceRoot) {
		if finding.Severity == runbook.SeveritySkip {
			return fmt.Errorf("role_invocation_failed: runbook sidecar %q: %s: %s", artifact.Path, finding.Code, finding.Detail)
		}
	}
	return nil
}
