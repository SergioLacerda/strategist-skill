// Package probe is the only source of "available" in the integration
// diagnostics. The documented JEV API has no health endpoint, so availability is
// the last successful synthetic call: a trivial, non-mission text with one
// question, cached with an expiry and invalidated by any change to the endpoint,
// model, data policy or capabilities. A missing, stale or invalidated
// observation is "unknown", never "available". A probe grants no authorization.
package probe

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
)

// SyntheticState is the whole text a probe sends. It contains nothing of a mission.
const SyntheticState = "Strategist connectivity check. This is a synthetic message with no mission content."

// TTL is how long a successful observation stays valid.
const TTL = 24 * time.Hour

// Availability is what the diagnostics may claim.
type Availability string

// Availability values.
const (
	Available   Availability = "available"
	Unavailable Availability = "unavailable"
	Unknown     Availability = "unknown"
)

// Record is one observation: metadata only, never a body or a credential.
type Record struct {
	Provider     string    `json:"provider"`
	Fingerprint  string    `json:"fingerprint"`
	At           time.Time `json:"at"`
	State        string    `json:"state"`
	Model        string    `json:"model,omitempty"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
	LatencyMS    int64     `json:"latency_ms,omitempty"`
}

// Fingerprint identifies what an observation is valid for. The credential
// reference is excluded on purpose: rotating a key must not change the identity.
func Fingerprint(p config.Provider) string {
	capabilities := make([]string, 0, len(p.AllowedCapabilities))
	for _, capability := range p.AllowedCapabilities {
		capabilities = append(capabilities, string(capability))
	}
	slices.Sort(capabilities)
	fields := slices.Clone(p.DataPolicy.AllowedFields)
	slices.Sort(fields)
	parts := []string{
		"strategist-integration-probe/v1", p.Endpoint, p.Model, fmt.Sprint(p.AllowModelAlias),
		strings.Join(capabilities, ","), strings.Join(fields, ","), fmt.Sprint(p.DataPolicy.MaxStateBytes),
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(strings.Join(parts, "\x00"))))
}

// Run sends the synthetic probe through the adapter once and records the
// outcome. A failure is recorded as its typed state, never as a provider body.
func Run(ctx context.Context, p config.Provider, adapter integration.Adapter, now func() time.Time) Record {
	started := now()
	record := Record{Provider: adapter.Identity().Provider, Fingerprint: Fingerprint(p), At: started, State: "ok"}
	result, err := adapter.Call(ctx, integration.Request{
		Correlation: integration.Correlation{Consumer: "probe"},
		Capability:  integration.CapHandoffValidate,
		State:       SyntheticState,
		Questions:   []integration.Question{{Key: "probe", Kind: integration.KindNoul, Instructions: "Is this message a synthetic connectivity check?"}},
	})
	record.LatencyMS = now().Sub(started).Milliseconds()
	if err != nil {
		state, ok := integration.StateOf(err)
		if !ok {
			state = integration.StateUnavailable
		}
		record.State = string(state)
		return record
	}
	record.Model = result.Model
	record.InputTokens, record.OutputTokens = result.Usage.InputTokens, result.Usage.OutputTokens
	return record
}

// Evaluate turns a stored observation into what may be claimed now, and until
// when that claim holds. Anything but a valid, matching observation is unknown.
func Evaluate(record Record, found bool, p config.Provider, now time.Time) (Availability, time.Time) {
	if !found || record.Fingerprint != Fingerprint(p) {
		return Unknown, time.Time{}
	}
	validUntil := record.At.Add(TTL)
	if now.After(validUntil) {
		return Unknown, validUntil
	}
	if record.State != "ok" {
		return Unavailable, validUntil
	}
	return Available, validUntil
}
