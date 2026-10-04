package domain

import "time"

// MissionInvocationSummary is the read-only projection of one request record
// for diagnostics. It deliberately carries no payload, input or nonce: a
// listing must be safe to paste into a ticket.
type MissionInvocationSummary struct {
	RequestID        string                  `json:"request_id"`
	MissionID        string                  `json:"mission_id"`
	Role             string                  `json:"role"`
	Slot             string                  `json:"slot"`
	State            MissionInvocationState  `json:"state"`
	CreatedAt        time.Time               `json:"created_at"`
	ExpiresAt        time.Time               `json:"expires_at"`
	Expired          bool                    `json:"expired"`
	ExecutionAdapter MissionExecutionAdapter `json:"execution_adapter,omitempty"`
}

// MissionInvocationListing is the result of listing request records. Skipped
// names the record files that could not be read, never their content.
type MissionInvocationListing struct {
	Requests []MissionInvocationSummary `json:"requests"`
	Skipped  []string                   `json:"skipped,omitempty"`
}
