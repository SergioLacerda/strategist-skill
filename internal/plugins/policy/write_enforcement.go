package policy

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// WriteDecision is the result of evaluating one candidate write path against
// a connector's EnforcementReport.
type WriteDecision struct {
	// Allowed is true only when the permission targetPath classifies to is
	// present in the connector's Enforceable set. There is no separate
	// "unknown" outcome — a permission the connector doesn't positively
	// claim to enforce is denied, not passed through.
	Allowed bool
	// Permission is the permission targetPath was classified as requiring.
	Permission domain.PluginPermission
	// Reason is a human-readable explanation suitable for a denial message.
	Reason string
}

// WriteScope contains onboarding-resolved roots. Roles remain portable and
// never encode repository directory names.
type WriteScope struct {
	AnalysisRoot       string
	DocumentationRoots []string
	RuntimeRoot        string
}

// defaultDocumentationRoot is used when active.yaml declares no
// documentation_roots.
const defaultDocumentationRoot = "docs"

// WriteScopeFromActive resolves cfg's base_path and documentation_roots
// against the project root that contains strategistRoot.
func WriteScopeFromActive(strategistRoot string, cfg domain.ActiveConfig) (WriteScope, error) {
	if cfg.BasePath == "" {
		return WriteScope{}, fmt.Errorf("active.yaml: base_path is empty")
	}
	roots := cfg.DocumentationRoots
	if len(roots) == 0 {
		roots = []string{defaultDocumentationRoot}
	}
	projectRoot := filepath.Dir(strategistRoot)
	scope := WriteScope{AnalysisRoot: filepath.Join(projectRoot, cfg.BasePath), RuntimeRoot: strategistRoot}
	for _, root := range roots {
		scope.DocumentationRoots = append(scope.DocumentationRoots, filepath.Join(projectRoot, root))
	}
	return scope, nil
}

// ClassifyWriteTarget maps a candidate write path to the PluginPermission
// that governs it, given the mission's configured analysis base_path (read
// from active.yaml — see domain.ActiveConfig.BasePath; this deliberately
// takes basePath as a parameter rather than hardcoding ".analysis", since a
// mission may configure a different base_path).
//
// Classification rules, most specific first:
//   - a path under basePath (e.g. ".analysis/refined/...") -> WriteAnalysis
//   - a path under "docs/" -> WriteDocs
//   - anything else -> WriteSource
//
// This mirrors the write-scope vocabulary roles/sniper.yaml already declares
// in prose (analysis vs. docs vs. source-code writes) so that a Go caller can
// evaluate a concrete path against it instead of relying on agent-contract
// text alone.
func ClassifyWriteTarget(basePath, targetPath string) domain.PluginPermission {
	return ClassifyWriteTargetInScope(WriteScope{AnalysisRoot: basePath, DocumentationRoots: []string{"docs"}}, targetPath)
}

// ClassifyWriteTargetInScope classifies a target against onboarding-resolved roots.
func ClassifyWriteTargetInScope(scope WriteScope, targetPath string) domain.PluginPermission {
	if underRoot(scope.AnalysisRoot, targetPath) {
		return domain.PluginPermissionWriteAnalysis
	}
	for _, root := range scope.DocumentationRoots {
		if underRoot(root, targetPath) {
			return domain.PluginPermissionWriteDocs
		}
	}
	return domain.PluginPermissionWriteSource
}

// EvaluateWrite reports whether a specific write to targetPath is
// enforceably allowed, given the mission's active.yaml base_path and the
// connector's EnforcementReport (as produced by a RuntimeConnector.Observe
// call — see internal/plugins/connectors.NativeRuntimeConnector.Observe).
//
// This is the Go-callable enforcement gate the K03 finding
// (.analysis/refined/20260830-skill-gaps-triage/analysis.md Cluster 1) found
// missing: policy.EvaluateGrant/EvaluateEnforcement already separate granted
// from enforceable permissions, but nothing consumed that distinction to
// deny a live write before it happened. Today Sniper's write_scope /
// approved_scope obligation (roles/sniper.yaml) is prose/agent-contract
// only — grep across the module turns up zero Go references to
// "write_scope" or "approved_scope" outside role/task YAML and docs, so
// there was no Go-side check at all for EvaluateWrite to have been missing
// from previously.
//
// EvaluateWrite denies by default: a write is allowed only when its
// classified permission is present in report.Enforceable. A permission that
// is merely Observed, or not mentioned at all, is denied — this matches
// NativeRuntimeConnector.Observe's own honest behavior of reporting only
// [ReadWorkspace, WriteAnalysis] as enforceable, which means a write
// targeting docs/ or a bare source file today is NOT enforceably allowed
// under the native connector, even though it may be nominally granted.
func EvaluateWrite(basePath, targetPath string, report EnforcementReport) WriteDecision {
	return EvaluateWriteInScope(WriteScope{AnalysisRoot: basePath, DocumentationRoots: []string{"docs"}}, targetPath, report)
}

// EvaluateWriteInScope applies connector enforcement to a target in a resolved scope.
func EvaluateWriteInScope(scope WriteScope, targetPath string, report EnforcementReport) WriteDecision {
	if underRoot(scope.RuntimeRoot, targetPath) {
		return WriteDecision{Allowed: false, Permission: domain.PluginPermissionWriteSource,
			Reason: fmt.Sprintf("write to %q is forbidden inside runtime root %q", targetPath, scope.RuntimeRoot)}
	}
	if isForbiddenRefinementPath(scope.DocumentationRoots, targetPath) {
		return WriteDecision{
			Allowed:    false,
			Permission: domain.PluginPermissionWriteDocs,
			Reason:     fmt.Sprintf("write to %q is forbidden for Strategist refinement artifacts; use <base_path>/refined/", targetPath),
		}
	}
	permission := ClassifyWriteTargetInScope(scope, targetPath)
	enforceable := permissionSet(report.Enforceable)
	if !enforceable[permission] {
		return WriteDecision{
			Allowed:    false,
			Permission: permission,
			Reason: fmt.Sprintf(
				"write to %q classified as permission %q is not in connector %q's enforceable set %v",
				targetPath, permission, report.ConnectorID, sortedPermissions(report.Enforceable),
			),
		}
	}
	return WriteDecision{
		Allowed:    true,
		Permission: permission,
		Reason: fmt.Sprintf(
			"write to %q classified as permission %q is enforceable by connector %q",
			targetPath, permission, report.ConnectorID,
		),
	}
}

func underRoot(root, target string) bool {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(target) == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

func isForbiddenRefinementPath(roots []string, target string) bool {
	for _, root := range roots {
		if underRoot(filepath.Join(root, "plans"), target) {
			return true
		}
	}
	return false
}

// IsForbiddenStrategistRefinementPath identifies repository planning paths that
// must never receive a Strategist Ranger or Archivist artifact. The check is
// intentionally independent of the configured base path: docs/plans is a
// forbidden target even when a connector advertises docs.write.
func IsForbiddenStrategistRefinementPath(targetPath string) bool {
	clean := filepath.ToSlash(filepath.Clean(targetPath))
	return clean == "docs/plans" || strings.HasPrefix(clean, "docs/plans/") ||
		strings.HasSuffix(clean, "/docs/plans") || strings.Contains(clean, "/docs/plans/")
}
