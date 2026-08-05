package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// handoffChallengeFile is the on-disk shape for --challenges.
type handoffChallengeFile struct {
	Challenges []handoffChallengeYAML `yaml:"challenges"`
}

type handoffChallengeYAML struct {
	ID                     string            `yaml:"id"`
	Type                   string            `yaml:"type"`
	SourceRefs             []string          `yaml:"source_refs"`
	Critical               bool              `yaml:"critical"`
	ExpectedClassification map[string]string `yaml:"expected_classification,omitempty"`
	ExpectedGateAllowed    *bool             `yaml:"expected_gate_allowed,omitempty"`
	ExpectedCounterfactual *bool             `yaml:"expected_counterfactual,omitempty"`
}

// handoffAckYAML is the on-disk shape for --ack.
type handoffAckYAML struct {
	ChallengeRefs         []string          `yaml:"challenge_refs"`
	UnderstoodRefs        []string          `yaml:"understood_refs"`
	Classifications       map[string]string `yaml:"classifications"`
	GateAllowed           *bool             `yaml:"gate_allowed,omitempty"`
	CounterfactualAnswers map[string]bool   `yaml:"counterfactual_answers,omitempty"`
}

// handoffPolicyYAML is the on-disk shape for --policy (optional override).
type handoffPolicyYAML struct {
	Enabled            bool     `yaml:"enabled"`
	Transition         string   `yaml:"transition"`
	RequiredTypes      []string `yaml:"required_types"`
	RequireAllCritical bool     `yaml:"require_all_critical"`
	MaxAttempts        int      `yaml:"max_attempts"`
	OnFailure          string   `yaml:"on_failure"`
	ForbiddenClaims    []string `yaml:"forbidden_claims,omitempty"`
}

func loadHandoffPolicy(path string) (handoff.Policy, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: operator-supplied policy file path, same trust level as --challenges/--ack
	if err != nil {
		return handoff.Policy{}, fmt.Errorf("read policy file: %w", err)
	}
	var y handoffPolicyYAML
	if err := yaml.Unmarshal(data, &y); err != nil {
		return handoff.Policy{}, fmt.Errorf("parse policy file: %w", err)
	}
	return handoff.Policy{
		Enabled:            y.Enabled,
		Transition:         y.Transition,
		RequiredTypes:      y.RequiredTypes,
		RequireAllCritical: y.RequireAllCritical,
		MaxAttempts:        y.MaxAttempts,
		OnFailure:          y.OnFailure,
		ForbiddenClaims:    y.ForbiddenClaims,
	}, nil
}

func loadHandoffChallenges(path string) ([]handoff.Challenge, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: operator-supplied challenges file path
	if err != nil {
		return nil, fmt.Errorf("read challenges file: %w", err)
	}
	var file handoffChallengeFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse challenges file: %w", err)
	}
	challenges := make([]handoff.Challenge, 0, len(file.Challenges))
	for _, c := range file.Challenges {
		challenges = append(challenges, handoff.Challenge{
			ID:                     c.ID,
			Type:                   c.Type,
			SourceRefs:             c.SourceRefs,
			Critical:               c.Critical,
			ExpectedClassification: c.ExpectedClassification,
			ExpectedGateAllowed:    c.ExpectedGateAllowed,
			ExpectedCounterfactual: c.ExpectedCounterfactual,
		})
	}
	return challenges, nil
}

func loadHandoffAck(path string) (handoff.Acknowledgment, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: operator-supplied acknowledgment file path
	if err != nil {
		return handoff.Acknowledgment{}, fmt.Errorf("read acknowledgment file: %w", err)
	}
	var y handoffAckYAML
	if err := yaml.Unmarshal(data, &y); err != nil {
		return handoff.Acknowledgment{}, fmt.Errorf("parse acknowledgment file: %w", err)
	}
	return handoff.Acknowledgment{
		ChallengeRefs:         y.ChallengeRefs,
		UnderstoodRefs:        y.UnderstoodRefs,
		Classifications:       y.Classifications,
		GateAllowed:           y.GateAllowed,
		CounterfactualAnswers: y.CounterfactualAnswers,
	}, nil
}

func printHandoffVerifyResult(cmd *cobra.Command, result handoff.Result) error {
	var b strings.Builder
	fmt.Fprintf(&b, "status: %s\n", result.Status)
	fmt.Fprintf(&b, "passed: %t\n", result.Passed)
	fmt.Fprintf(&b, "critical_failures: %d\n", result.CriticalFailures)
	appendHandoffVerifyList(&b, "missing_refs", result.MissingRefs)
	appendHandoffVerifyList(&b, "missing_challenges", result.MissingChallenges)
	appendHandoffVerifyList(&b, "misclassified_refs", result.MisclassifiedRefs)
	appendHandoffVerifyList(&b, "counterfactual_mismatches", result.CounterfactualMismatches)
	appendHandoffVerifyList(&b, "forbidden_claim_violations", result.ForbiddenClaimViolations)
	if result.GateMismatch {
		b.WriteString("gate_mismatch: true\n")
	}
	if result.NextAction != "" {
		fmt.Fprintf(&b, "next_action: %s\n", result.NextAction)
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), b.String()); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func appendHandoffVerifyList(b *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(b, "%s: %v\n", label, values)
}

func recordHandoffVerify(cmd *cobra.Command, opts handoffVerifyOptions, result handoff.Result) error {
	root, err := resolveMetricsActionRoot(cmd, "handoff verify", opts.Root)
	if err != nil {
		return err
	}
	rec := telemetry.ChallengeRecord{
		MissionID:                opts.MissionID,
		Transition:               opts.Transition,
		Attempt:                  opts.Attempt,
		Timestamp:                time.Now().UTC().Format(time.RFC3339),
		Status:                   result.Status,
		Passed:                   result.Passed,
		MissingRefs:              result.MissingRefs,
		MissingChallenges:        result.MissingChallenges,
		MisclassifiedRefs:        result.MisclassifiedRefs,
		GateMismatch:             result.GateMismatch,
		CounterfactualMismatches: result.CounterfactualMismatches,
		ForbiddenClaimViolations: result.ForbiddenClaimViolations,
		CriticalFailures:         result.CriticalFailures,
	}
	if err := telemetry.AppendHandoffChallenge(telemetry.HandoffChallengeHistoryPath(root), rec); err != nil {
		return fmt.Errorf("record handoff challenge: %w", err)
	}
	return nil
}
