package jeweler

import (
	"fmt"
	"strings"
	"time"
)

// QueryRequest carries a bounded, read-only provider query.
type QueryRequest struct {
	Requester     string   `json:"requester"`
	MissionID     string   `json:"mission_id"`
	CorrelationID string   `json:"correlation_id"`
	Root          string   `json:"root"`
	Intent        string   `json:"intent,omitempty"`
	Concepts      []string `json:"concepts,omitempty"`
	Symbols       []string `json:"symbols,omitempty"`
	Components    []string `json:"components,omitempty"`
	Limit         int      `json:"limit"`
	FreshnessTTL  string   `json:"freshness_ttl,omitempty"`
}

// Validate checks the requester, mission identity, limit, and freshness bound.
func (r QueryRequest) Validate() error {
	if err := validateQueryIdentity(r); err != nil {
		return err
	}
	if err := validateQueryLimit(r.Limit); err != nil {
		return err
	}
	return validateFreshnessTTL(r.FreshnessTTL)
}

func validateQueryIdentity(r QueryRequest) error {
	if r.Requester != "ranger" && r.Requester != "archivist" {
		return fmt.Errorf("jeweler: requester %q is not allowed", r.Requester)
	}
	for name, value := range map[string]string{"mission_id": r.MissionID, "correlation_id": r.CorrelationID, "root": r.Root} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("jeweler: %s is required", name)
		}
	}
	return nil
}

func validateQueryLimit(limit int) error {
	if limit <= 0 || limit > 100 {
		return fmt.Errorf("jeweler: limit must be between 1 and 100")
	}
	return nil
}

func validateFreshnessTTL(value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.ParseDuration(value); err != nil {
		return fmt.Errorf("jeweler: freshness_ttl: %w", err)
	}
	return nil
}
