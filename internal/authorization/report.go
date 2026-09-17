// Package authorization composes the independent evidence dimensions used by
// the CLI authorization boundary. It deliberately does not intercept writes.
package authorization

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
)

// ReportSchemaVersion identifies the stable JSON schema emitted by Report.JSON.
const ReportSchemaVersion = "strategist-authorization-report/v1"

// ErrDenied indicates that a required authorization dimension rejected the target.
var ErrDenied = errors.New("authorization denied")

// ErrBlocked indicates that authorization evidence or a required gate is unavailable.
var ErrBlocked = errors.New("authorization blocked")

// ErrStale indicates that the compiled runtime artifact is stale.
var ErrStale = errors.New("authorization stale")

// Dimension is one independently evaluated authorization input.
type Dimension struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	EvidenceState string `json:"evidence_state,omitempty"`
	ReasonCode    string `json:"reason_code,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Required      bool   `json:"required"`
}

// Report is the complete machine-readable authorization decision.
type Report struct {
	SchemaVersion string      `json:"schema_version"`
	GeneratedAt   string      `json:"generated_at"`
	Root          string      `json:"runtime_root"`
	Target        string      `json:"target"`
	MissionID     string      `json:"mission_id,omitempty"`
	Role          string      `json:"role,omitempty"`
	Provider      string      `json:"provider,omitempty"`
	Permission    string      `json:"permission,omitempty"`
	Decision      string      `json:"decision"`
	ReasonCode    string      `json:"reason_code"`
	ExitClass     string      `json:"exit_class"`
	Dimensions    []Dimension `json:"dimensions"`
	Limitations   []string    `json:"limitations,omitempty"`
}

// Request contains the runtime and gate inputs used to build an authorization report.
type Request struct {
	Root          string
	Target        string
	MissionID     string
	ApprovalGate  string
	ExecutionGate string
}

// Build evaluates all authorization dimensions for a candidate target.
func Build(request Request) (Report, error) {
	report := Report{
		SchemaVersion: ReportSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Root:          request.Root,
		Target:        request.Target,
		MissionID:     request.MissionID,
		Decision:      "denied",
		ExitClass:     "denied",
		Dimensions:    []Dimension{},
	}
	if request.Target == "" {
		return report, fmt.Errorf("authorization: target is required")
	}

	active, err := loadActive(request.Root)
	if err != nil {
		report.Dimensions = append(report.Dimensions, Dimension{Name: "runtime", Status: "blocked", EvidenceState: "failed", ReasonCode: "runtime_unavailable", Detail: err.Error(), Required: true})
		return finish(report), fmt.Errorf("authorization: %w: %v", ErrBlocked, err)
	}
	report.Dimensions = append(report.Dimensions, Dimension{Name: "runtime", Status: "ready", EvidenceState: "static", ReasonCode: "runtime_config_verified", Detail: request.Root, Required: true})
	if artifactDimension := compiledArtifactEvidence(request.Root); artifactDimension.Name != "" {
		report.Dimensions = append(report.Dimensions, artifactDimension)
	}
	roles, roleErr := loadRoleMap(request.Root)
	if roleErr == nil {
		report.Role = roles["execution"]
	}
	report.Provider = active.Slots["execution"]

	bindingDimension := bindingEvidence(request.Root, active)
	report.Dimensions = append(report.Dimensions, bindingDimension)

	gateDimension := gateEvidence(request.ApprovalGate, request.ExecutionGate)
	report.Dimensions = append(report.Dimensions, gateDimension)

	connector := connectors.NativeRuntimeConnector{ConnectorID: "strategist-native", ConnectorAPIVersion: "strategist-connector-api/1", EnforcementObservable: true}
	observation := connector.Observe(context.Background(), domain.InstalledInstance{ID: "sniper", State: "active"})
	projectRoot := filepath.Dir(request.Root)
	documentationRoots := active.DocumentationRoots
	if len(documentationRoots) == 0 {
		documentationRoots = []string{"docs"}
	}
	scope := policy.WriteScope{AnalysisRoot: filepath.Join(projectRoot, active.BasePath), RuntimeRoot: request.Root}
	for _, root := range documentationRoots {
		scope.DocumentationRoots = append(scope.DocumentationRoots, filepath.Join(projectRoot, root))
	}
	permission := policy.ClassifyWriteTargetInScope(scope, filepath.Join(projectRoot, request.Target))
	report.Permission = string(permission)
	decision := policy.EvaluateWriteInScope(scope, filepath.Join(projectRoot, request.Target), observation.Enforcement)
	status := "denied"
	if decision.Allowed {
		status = "allowed"
	}
	report.Dimensions = append(report.Dimensions, Dimension{Name: "target", Status: status, EvidenceState: "observed", ReasonCode: targetReason(decision), Detail: decision.Reason, Required: true})

	// The native connector intentionally cannot prove invocation of external
	// providers. Keep that limitation visible and non-required for the CLI
	// before-write decision; callers must not mistake allowed target permission
	// for live provider authorization.
	report.Dimensions = append(report.Dimensions, Dimension{Name: "live_provider", Status: "unknown", EvidenceState: "unverified", ReasonCode: "live_invocation_unverified", Detail: "CLI authorization does not invoke external providers", Required: false})
	report = finish(report)
	return report, authorizationError(report)
}
