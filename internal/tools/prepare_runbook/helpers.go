package preparerunbook

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
)

type artifact struct {
	path  string
	text  string
	front map[string]any
}

func inspectEvidence(text string) Evidence {
	lower := strings.ToLower(text)
	return Evidence{
		Procedure:       hasAny(lower, "procedure", "resolution steps", "recovery sequence", "steps", "runbook"),
		Reproducible:    hasAny(lower, "go test", "make ", "npm ", "curl ", "command", "reproduce", "reproducible", "```"),
		ExpectedOutputs: hasAny(lower, "expected output", "expected result", "expected behavior", "verification", "assert", "result"),
		StopConditions:  hasAny(lower, "stop condition", "stop when", "halt", "do not proceed", "abort", "blocking"),
		Provenance:      hasAny(lower, "provenance", "source", "evidence", "digest", "ref:", "reference"),
		Scope:           hasAny(lower, "scope", "affected", "allowed", "forbidden", "path", "boundary"),
	}
}

func buildCandidate(in Input, artifacts []artifact, text string, evidence Evidence) runbook.Runbook {
	objective := firstFrontString(artifacts, "mission_objective", "objective")
	if objective == "" {
		objective = firstHeadingText(text, "objective", "mission objective")
	}
	if objective == "" {
		objective = "Provide a repeatable resolution procedure for " + in.MissionID
	}
	applies := firstFrontStrings(artifacts, "applies_when", "signals")
	if len(applies) == 0 {
		applies = []string{"the refined mission presents a repeatable operational symptom"}
	}
	rbType := runbook.RunbookTypeOperational
	if strings.Contains(strings.ToLower(text), "analytical runbook") {
		rbType = runbook.RunbookTypeAnalytical
	}
	id := "oa-" + slug(in.MissionID) + "-" + shortDigest(in.PackageDigest)
	rb := runbook.Runbook{
		SchemaVersion: "2", RunbookID: id, RunbookType: rbType,
		SourceDoc:   filepath.ToSlash(filepath.Join("refined", in.MissionID, "analysis.md")),
		AppliesWhen: applies, Signals: signalNames(text), Objective: objective,
		Preconditions: matchingLines(text, "precondition", "before", "requires"),
		Analysis:      matchingLines(text, "root cause", "reason", "finding"),
		Verification:  matchingLines(text, "verify", "verification", "expected output", "expected result", "check"),
		Checks:        []runbook.Check{{ID: "reproducible-procedure", Level: runbook.LevelMandatory}},
		Metadata:      runbook.Metadata{Version: 1, Owner: "opportunity-attack"},
	}
	if evidence.StopConditions {
		rb.DecisionGates = []runbook.DecisionGate{{ID: "stop-condition", Statement: "Stop when the stated stop condition is reached or evidence is insufficient."}}
	}
	if len(rb.Verification) == 0 {
		rb.Verification = []string{"Verify the expected output against the cited source evidence."}
	}
	return rb
}

func providerFields(rb runbook.Runbook, comparison Comparison) map[string]string {
	return map[string]string{"objective": rb.Objective, "applies_when": strings.Join(rb.AppliesWhen, "; "), "preconditions": strings.Join(rb.Preconditions, "; "), "verification": strings.Join(rb.Verification, "; "), "comparison": string(comparison.Classification)}
}

func provenance(artifacts []artifact) []Provenance {
	result := make([]Provenance, 0, len(artifacts))
	for _, a := range artifacts {
		raw, err := os.ReadFile(a.path)
		if err != nil {
			continue
		}
		result = append(result, Provenance{Path: a.path, Role: filepath.Base(a.path), Digest: fmt.Sprintf("sha256:%x", sha256.Sum256(raw))})
	}
	return result
}
