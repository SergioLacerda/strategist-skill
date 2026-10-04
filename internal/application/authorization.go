package application

import "fmt"

// AuthorizationDimension is one independent authorization evidence row.
type AuthorizationDimension struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	EvidenceState string `json:"evidence_state,omitempty"`
	ReasonCode    string `json:"reason_code,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Required      bool   `json:"required"`
}

// AuthorizationReport is the application-facing plugin authorization result.
type AuthorizationReport struct {
	SchemaVersion string                   `json:"schema_version"`
	GeneratedAt   string                   `json:"generated_at"`
	Root          string                   `json:"runtime_root"`
	Target        string                   `json:"target"`
	MissionID     string                   `json:"mission_id,omitempty"`
	Role          string                   `json:"role,omitempty"`
	Provider      string                   `json:"provider,omitempty"`
	Permission    string                   `json:"permission,omitempty"`
	Decision      string                   `json:"decision"`
	ReasonCode    string                   `json:"reason_code"`
	ExitClass     string                   `json:"exit_class"`
	Dimensions    []AuthorizationDimension `json:"dimensions"`
	Limitations   []string                 `json:"limitations,omitempty"`
}

// AuthorizationRequest is the application-owned input to plugin authorization.
type AuthorizationRequest struct {
	Root          string
	Target        string
	MissionID     string
	ApprovalGate  string
	ExecutionGate string
}

// Authorize delegates concrete evidence collection to a plugin authorization
// adapter while keeping request validation and orchestration at the application
// boundary.
func Authorize(request AuthorizationRequest, build func(AuthorizationRequest) (AuthorizationReport, error)) (AuthorizationReport, error) {
	if request.Target == "" {
		return AuthorizationReport{}, fmt.Errorf("authorization target is required")
	}
	if build == nil {
		return AuthorizationReport{}, fmt.Errorf("authorization adapter is unavailable")
	}
	return build(request)
}
