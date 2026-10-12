package jeweler

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// CommandRunner is injected in tests and keeps command execution explicit.
type CommandRunner func(context.Context, string, ...string) ([]byte, error)

// OSCommandRunner executes an explicitly bound provider executable.
func OSCommandRunner(ctx context.Context, executable string, args ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, executable, args...).Output() //nolint:gosec // executable is an explicit operator binding
	if err != nil {
		return nil, fmt.Errorf("run provider command: %w", err)
	}
	return output, nil
}

// CLIProvider adapts the standalone skill-for-hire CLI. It never discovers a
// binary by PATH and never substitutes the in-repo treasure-chest package.
type CLIProvider struct {
	Executable string
	Identity   ProviderIdentity
	// Evidence is the host admission result for static package metadata and
	// the delegate. Probe and health are collected from the provider status
	// command and are never assumed from this field.
	Evidence EvidenceVector
	Run      CommandRunner
}

type externalStatusResponse struct {
	Provider      string `json:"Provider"`
	Version       string `json:"Version"`
	SchemaVersion int    `json:"SchemaVersion"`
	Healthy       bool   `json:"Healthy"`
}

func (p CLIProvider) runner() (CommandRunner, error) {
	if p.Executable == "" {
		return nil, fmt.Errorf("jeweler: external provider executable is not configured")
	}
	if err := p.Identity.Validate(); err != nil {
		return nil, err
	}
	if p.Run != nil {
		return p.Run, nil
	}
	return OSCommandRunner, nil
}

// Readiness probes the external provider and evaluates its admission evidence.
func (p CLIProvider) Readiness(ctx context.Context, root string) (Readiness, error) {
	run, err := p.runner()
	if err != nil {
		return Readiness{Status: "blocked", Reason: "provider_binding_invalid"}, err
	}
	readiness, static, done := prepareReadiness(p.Evidence)
	if done {
		return readiness, nil
	}
	return p.probeReadiness(ctx, root, run, static)
}

func prepareReadiness(static EvidenceVector) (Readiness, EvidenceVector, bool) {
	if static.Package == "" && static.Identity == "" && static.Provenance == "" && static.API == "" && static.Delegate == "" {
		return ReadinessFromEvidence(static), static, true
	}
	if static.Probe != "" || static.Health != "" {
		return Readiness{Status: "blocked", Reason: "live_evidence_not_host_collected", Evidence: static}, static, true
	}
	static.Probe = EvidenceUnknown
	static.Health = EvidenceUnknown
	readiness := ReadinessFromEvidence(static)
	return readiness, static, readiness.Status != "ready" && readiness.Reason != "probe_unverified"
}

func (p CLIProvider) probeReadiness(ctx context.Context, root string, run CommandRunner, static EvidenceVector) (Readiness, error) {
	data, err := run(ctx, p.Executable, "status", "--root", root, "--json")
	if err != nil {
		static.Probe = EvidenceUnavailable
		return ReadinessFromEvidence(static), err
	}
	var status externalStatusResponse
	if err := json.Unmarshal(data, &status); err != nil {
		static.Probe = EvidenceMalformed
		return ReadinessFromEvidence(static), err
	}
	if !statusMatchesIdentity(status, p.Identity) {
		static.Identity = EvidenceBlocked
		return ReadinessFromEvidence(static), nil
	}
	static.Probe = EvidenceCertified
	if !status.Healthy {
		static.Health = EvidenceFailed
		return ReadinessFromEvidence(static), nil
	}
	static.Health = EvidenceCertified
	return ReadinessFromEvidence(static), nil
}

func statusMatchesIdentity(status externalStatusResponse, identity ProviderIdentity) bool {
	return status.Provider == identity.Package && status.Version == identity.Version && status.SchemaVersion == KnowledgeAPIVersion
}

// Maintain invokes an explicitly authorized provider maintenance operation.
func (p CLIProvider) Maintain(ctx context.Context, req MaintenanceRequest) (MaintenanceResult, error) {
	if err := req.Validate(); err != nil {
		return MaintenanceResult{}, err
	}
	if req.Operation != "index" && req.Operation != "probe" && req.Operation != "health" {
		return MaintenanceResult{}, fmt.Errorf("external CLI does not expose maintenance operation %q", req.Operation)
	}
	run, err := p.runner()
	if err != nil {
		return MaintenanceResult{}, err
	}
	command := maintenanceCommand(req.Operation)
	if _, err := run(ctx, p.Executable, command, "--root", req.Root, "--json"); err != nil {
		return MaintenanceResult{Operation: req.Operation, Readiness: Readiness{Status: "failed", Reason: "maintenance_command_failed"}}, err
	}
	readiness, err := p.Readiness(ctx, req.Root)
	if err != nil {
		return MaintenanceResult{Operation: req.Operation, Readiness: readiness}, err
	}
	if !readiness.Ready() {
		return MaintenanceResult{Operation: req.Operation, Readiness: readiness}, fmt.Errorf("maintenance command completed but provider is not ready: %s", readiness.Reason)
	}
	readiness.Reason = "maintenance_command_completed"
	return MaintenanceResult{Operation: req.Operation, Readiness: readiness}, nil
}

func maintenanceCommand(operation string) string {
	if operation == "index" {
		return "prepare"
	}
	return "status"
}
