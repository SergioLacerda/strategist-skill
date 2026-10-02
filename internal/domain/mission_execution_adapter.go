package domain

import "fmt"

// MissionExecutionAdapter names which Strategist-owned path produced a Ranked
// Embedded completion. It is committed by Strategist control flow, never read
// from a completion, and says nothing about parent capability isolation.
type MissionExecutionAdapter string

const (
	// ExecutionAdapterCurrentHost is the default: the managed current-host agent
	// returns the completion; no child process was launched by Strategist.
	ExecutionAdapterCurrentHost MissionExecutionAdapter = "current_host_adapter"
	// ExecutionAdapterCodexChild is a Codex child launched by `mission invoke --host codex`.
	ExecutionAdapterCodexChild MissionExecutionAdapter = "codex_child"
	// ExecutionAdapterClaudeChild is a Claude child launched by `mission invoke --host claude`.
	ExecutionAdapterClaudeChild MissionExecutionAdapter = "claude_child"
	// ExecutionAdapterCurrentHostUnverified is only how a record written before
	// the field existed is read. It is never committed, and never upgraded to a child mode.
	ExecutionAdapterCurrentHostUnverified MissionExecutionAdapter = "current_host_adapter_unverified"
)

// Committable reports whether Strategist control flow may commit the mode.
func (a MissionExecutionAdapter) Committable() bool {
	return a == ExecutionAdapterCurrentHost || a.IsChild()
}

// IsChild reports whether Strategist launched a child process for this mode.
func (a MissionExecutionAdapter) IsChild() bool {
	return a == ExecutionAdapterCodexChild || a == ExecutionAdapterClaudeChild
}

// Known reports whether the value belongs to the closed vocabulary.
func (a MissionExecutionAdapter) Known() bool {
	return a.Committable() || a == ExecutionAdapterCurrentHostUnverified
}

// ChildAdapterForHost maps the closed `--host` vocabulary to its child mode.
func ChildAdapterForHost(host string) (MissionExecutionAdapter, error) {
	switch host {
	case "codex":
		return ExecutionAdapterCodexChild, nil
	case "claude":
		return ExecutionAdapterClaudeChild, nil
	default:
		return "", fmt.Errorf("unsupported host %q (expected codex or claude)", host)
	}
}
