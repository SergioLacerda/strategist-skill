package handoffconsumer

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
)

// PolicyRow is one check of the integration layer: what the Strategist policy
// does with a scripted provider answer. No call is made; the answers are known,
// so a failure points at the policy and never at the provider.
type PolicyRow struct {
	Name   string
	Pass   bool
	Detail string
}

func scripted(choice string, confidence float64, accept float64) integration.Result {
	return integration.Result{Model: "jev-1.13.0", Answers: map[string]integration.Answer{AggregateKey: {
		Kind: integration.KindChoice, Choice: choice, Confidence: confidence, HasConfidence: true,
		Probabilities: map[string]float64{OptionAccept: accept, OptionReject: 1 - accept},
	}}}
}

func row(name string, pass bool, format string, args ...any) PolicyRow {
	return PolicyRow{Name: name, Pass: pass, Detail: fmt.Sprintf(format, args...)}
}

// PolicyMatrix evaluates the approval rule, the never-approves cases, the
// fallback of every failing state, the delegable checks and the resolution of a
// pinned calibration, with scripted answers and a scripted resolver.
func PolicyMatrix(now time.Time) []PolicyRow {
	rows := approvalRows()
	rows = append(rows, fallbackRows()...)
	rows = append(rows, delegationRows()...)
	return append(rows, calibrationRows(now)...)
}

func approvalRows() []PolicyRow {
	at, threshold := TransitionArchivistToGate, config.DefaultThreshold
	approves := func(r integration.Result) bool { return Evaluate(at, r, threshold).Approved }
	noAggregate := integration.Result{Answers: map[string]integration.Answer{"scope_bounded": {Kind: integration.KindNoul, Noul: 1}}}
	return []PolicyRow{
		row("policy approves above the threshold", approves(scripted(OptionAccept, 0.70, 0.96)), "accept with P(accept) 0.96 over %.2f", threshold),
		row("policy never approves at exactly the threshold", !approves(scripted(OptionAccept, 0.9, threshold)), "strictly above is the rule"),
		row("policy never approves below the threshold", !approves(scripted(OptionAccept, 0.9, 0.79)), "P(accept) 0.79"),
		row("policy never approves a reject however confident", !approves(scripted(OptionReject, 0.99, 0.01)), "reject at 99%% is 1%% confidence of acceptance"),
		row("policy never approves a missing aggregate", !approves(noAggregate), "a noul answer has no confidence"),
		row("policy reads P(accept) before the documented confidence", approves(scripted(OptionAccept, 0.30, 0.96)), "confidence 0.30 but P(accept) 0.96"),
	}
}

func fallbackRows() []PolicyRow {
	binding := integration.Binding{Provider: "jev", Version: "1", Model: "jev-1.13.0", Consumer: "handoff", Capability: integration.CapHandoffValidate}
	states := failingStates()
	rows := make([]PolicyRow, 0, len(states)+1)
	for _, state := range states {
		d := policy.Select(state, binding)
		rows = append(rows, row("policy falls back to main on "+string(state), d.Effective == policy.PathMain && d.FellBack, "effective %s", d.Effective))
	}
	d := policy.Select(integration.StateBindingIntegrity, binding)
	return append(rows, row("policy flags a binding integrity failure as suspect", d.Suspect, "main serves the call and the failure is reported"))
}

// failingStates are every state but the configured choice to be disabled.
func failingStates() []integration.State {
	var out []integration.State
	for _, state := range integration.States() {
		if state != integration.StateDisabled {
			out = append(out, state)
		}
	}
	return out
}

func delegationRows() []PolicyRow {
	delegable := handoff.DelegableChecks()
	return []PolicyRow{
		row("policy delegates only input_output_conformance", len(delegable) == 1 && delegable[0] == handoff.CheckInputOutputConformance, "%v", delegable),
		row("policy never delegates integrity, gate or authorization", len(handoff.NeverDelegatedChecks()) == 5, "%v", handoff.NeverDelegatedChecks()),
	}
}

func calibrationRows(now time.Time) []PolicyRow {
	binding := calibration.Binding{Model: "jev-1.13.0", Profile: ProfileID, ProfileVersion: ProfileVersion(calibration.BoundaryPackage)}
	valid := scriptedRecord(0.60, binding, now)
	use := func(record calibration.Record, model string, at time.Time) CalibrationUse {
		c := &Consumer{Provider: config.Provider{CalibrationID: record.ID}, Now: func() time.Time { return at },
			Calibrations: func(string) (calibration.Record, error) { return record, nil }}
		return c.calibrationFor(TransitionArchivistToGate, model)
	}
	other := binding
	other.ProfileVersion = "00000000"
	return []PolicyRow{
		row("pinned calibration applies when valid", use(valid, "jev-1.13.0", now).Applied, "threshold %.2f", 0.60),
		row("pinned calibration is ignored for another model", !use(valid, "jev-2.0.0", now).Applied, "reason %s", use(valid, "jev-2.0.0", now).Reason),
		row("pinned calibration is ignored for another profile", !use(scriptedRecord(0.6, other, now), "jev-1.13.0", now).Applied, "the questions changed"),
		row("pinned calibration is ignored when too old", !use(valid, "jev-1.13.0", now.Add(120*24*time.Hour)).Applied, "older than %d days", 90),
		row("pinned calibration is ignored for another profile id", !profileUse(valid, now).Applied, "a v1 calibration never serves a v2 provider"),
		row("no pin means the configured threshold", !(&Consumer{Provider: config.Provider{}}).calibrationFor(TransitionArchivistToGate, "m").Applied, "never a silent default"),
	}
}

func profileUse(record calibration.Record, now time.Time) CalibrationUse {
	c := &Consumer{Provider: config.Provider{CalibrationID: record.ID, Profile: calibration.ProfileV2}, Now: func() time.Time { return now },
		Calibrations: func(string) (calibration.Record, error) { return record, nil }}
	return c.calibrationFor(TransitionArchivistToGate, "jev-1.13.0")
}

func scriptedRecord(threshold float64, binding calibration.Binding, now time.Time) calibration.Record {
	record, err := calibration.NewRecord(calibration.Report{Status: calibration.StatusRecommended, Threshold: &threshold},
		calibration.Meta{Provider: "jev", Model: binding.Model, Profile: binding.Profile, ProfileVersion: binding.ProfileVersion, Now: now})
	if err != nil {
		return calibration.Record{}
	}
	return record
}
