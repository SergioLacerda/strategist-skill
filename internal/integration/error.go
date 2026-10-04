package integration

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const maxReasonLength = 160

// Correlation ties an error to the mission and consumer that made the call.
type Correlation struct {
	MissionID string
	Consumer  string
	Provider  string
}

// Error is the only error shape the mechanism returns. Reason is sanitized and
// never carries a provider body or a credential.
type Error struct {
	State State
	Correlation
	Reason string
}

// NewError builds a sanitized, correlated error.
func NewError(state State, correlation Correlation, reason string) *Error {
	return &Error{State: state, Correlation: correlation, Reason: Sanitize(reason)}
}

func (e *Error) Error() string {
	return fmt.Sprintf("integration %s %s: %s", e.Provider, e.State, e.Reason)
}

// StateOf reports the state of an integration error anywhere in err's chain.
func StateOf(err error) (State, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target.State, true
	}
	return "", false
}

// Sanitize removes control characters, redacts each known secret and bounds the
// length, so a reason can be logged or persisted as is.
func Sanitize(reason string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			reason = strings.ReplaceAll(reason, secret, "[redacted]")
		}
	}
	reason = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, reason)
	if len(reason) > maxReasonLength {
		reason = reason[:maxReasonLength]
	}
	return reason
}

// WithCorrelation fills the empty correlation fields of an integration error
// so a lower layer that did not know the mission still ends up correlated.
// Any other error is returned unchanged.
func WithCorrelation(err error, correlation Correlation) error {
	var target *Error
	if !errors.As(err, &target) {
		return err
	}
	enriched := *target
	if enriched.MissionID == "" {
		enriched.MissionID = correlation.MissionID
	}
	if enriched.Consumer == "" {
		enriched.Consumer = correlation.Consumer
	}
	if enriched.Provider == "" {
		enriched.Provider = correlation.Provider
	}
	return &enriched
}
