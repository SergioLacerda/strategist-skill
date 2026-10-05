package handoffconsumer

import (
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
)

func hintKeys(hints []Hint) []string {
	if len(hints) == 0 {
		return nil
	}
	keys := make([]string, 0, len(hints))
	for _, hint := range hints {
		keys = append(keys, hint.Criterion)
	}
	return keys
}

// Reuse restates the pre-gate reading of the same revision for the Archivist to
// Sniper evaluation, so the provider is not asked twice. It reuses only an
// intact, answered record of this mission and package digest from the same
// provider and pinned model, and it applies the current local threshold to the
// recorded confidence: the rule stays local even when the answer is reused.
func (c *Consumer) Reuse(request Request) (Report, bool) {
	record, ok := c.reusableRecord(request)
	if !ok {
		return Report{}, false
	}
	use := c.calibrationFor(request.Transition, record.Model)
	report := Report{
		Reused: true, Confidence: record.Confidence, Threshold: c.threshold(use), Calibration: use,
		Decision: policy.Decision{Preferred: policy.PathJEV, Effective: policy.PathJEV, Binding: record.Binding},
	}
	c.restate(&report, record, request.Subject)
	c.record(request, report, mechanism.Outcome{Decision: report.Decision}, 0)
	return report, true
}

// reusableRecord finds the intact pre-gate reading this evaluation may restate.
func (c *Consumer) reusableRecord(request Request) (Record, bool) {
	if request.Transition != handoff.TransitionArchivistToSniper || !c.Provider.Enabled {
		return Record{}, false
	}
	record, found, err := c.Ledger.Latest(request.MissionID, TransitionArchivistToGate, request.Subject)
	if err != nil || !found || record.Provider != c.Name || !c.modelStillBound(record.Model) || (record.Shadow && !c.Provider.Shadow) {
		return Record{}, false
	}
	return record, true
}

// restate applies the current local rule to the recorded confidence.
func (c *Consumer) restate(report *Report, record Record, subject string) {
	if c.Provider.Shadow {
		report.Status, report.WouldApprove = StatusShadow, record.Confidence > report.Threshold && len(record.Hints) == 0
		return
	}
	if record.Confidence > report.Threshold && len(record.Hints) == 0 {
		report.Status = StatusApproved
		report.Delegation = c.delegationFrom(record, subject, report.Threshold, report.Calibration)
		return
	}
	report.Status = StatusSignal
	for _, key := range record.Hints {
		report.Hints = append(report.Hints, Hint{Criterion: key})
	}
}

func (c *Consumer) modelStillBound(resolved string) bool {
	return c.Provider.AllowModelAlias || resolved == c.Provider.Model
}

func (c *Consumer) delegationFrom(record Record, subject string, threshold float64, use CalibrationUse) *handoff.Delegation {
	return &handoff.Delegation{
		Provider: c.Name, Model: record.Model, BindingDigest: record.Binding,
		Capability: string(integration.CapHandoffValidate), Criterion: AggregateKey, Subject: subject,
		Checks: handoff.DelegableChecks(), Confidence: record.Confidence, Threshold: threshold,
		InputTokens: record.InputTokens, OutputTokens: record.OutputTokens, CalibrationID: appliedID(use),
	}
}
