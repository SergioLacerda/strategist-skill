package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// LifecycleDependencies injects the runtime-specific persistence, path
// resolution and output encoding used by start, status, submit and context.
type LifecycleDependencies struct {
	RootFlag          string
	RequireMissionID  func(string) error
	ResolveBasePath   func(string) (string, string, error)
	RequireNoExisting func(string, string) error
	Save              func(string, domain.MissionEngineStatus) error
	Load              func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error)
	InitiativeStart   func(string, string) error
	WriteResult       func(*cobra.Command, bool, any) error
	// Lock, when set, serializes a mission's read-modify-write critical
	// section (RequireNoExisting/Load through Save) against concurrent CLI
	// invocations for the same mission id (ADR-0057 § D2 — mission state
	// concurrency). Nil is a valid, backward-compatible no-op: fn runs
	// unguarded, which is what every pre-existing caller and test fake
	// already did before this field existed.
	Lock func(root, missionID string, fn func() error) error
	// ADRCanonicalPath reads active.yaml#adr.canonical_path for the
	// accept-side-quest command. Nil means no canonical path is configured.
	ADRCanonicalPath func(root string) (string, error)
	// TelemetrySink selects the route-resolution sink. Nil keeps the adapter
	// compatible with callers that only need durable route history.
	TelemetrySink func() telemetry.EventSink
}

// withMissionLock runs fn under deps.Lock when one is configured, or
// unguarded otherwise. See LifecycleDependencies.Lock's doc comment for why
// nil is a safe default.
func withMissionLock(deps LifecycleDependencies, root, missionID string, fn func() error) error {
	if deps.Lock == nil {
		return fn()
	}
	return deps.Lock(root, missionID, fn)
}

// lifecycleFlags holds the --root/--mission-id/--json values shared by the
// lifecycle subcommands.
type lifecycleFlags struct {
	root, missionID string
	asJSON          bool
}

func bindLifecycleFlags(cmd *cobra.Command, deps LifecycleDependencies) *lifecycleFlags {
	f := &lifecycleFlags{}
	cmd.Flags().StringVar(&f.root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().StringVar(&f.missionID, "mission-id", "", "mission identifier (required)")
	cmd.Flags().BoolVar(&f.asJSON, "json", false, "emit machine-readable JSON")
	return f
}

// NewStart builds `mission start`.
func NewStart(deps LifecycleDependencies) *cobra.Command {
	cmd := &cobra.Command{Use: "start", Short: "Start a mission"}
	f := bindLifecycleFlags(cmd, deps)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunStart(cmd, deps, f.root, f.missionID, f.asJSON) }
	return cmd
}

// RunStart creates and persists a new mission through domain.StartMission.
//
// The existence check, INITIATIVE consultation, engine creation and save all
// run inside one lock acquisition (deps.Lock, when configured), so two
// concurrent `mission start` calls for the same mission id cannot both pass
// RequireNoExisting before either has saved — the check-then-create race
// ADR-0057 § D2 closes alongside RunSubmit's own read-modify-write.
func RunStart(cmd *cobra.Command, deps LifecycleDependencies, rootInput, missionID string, asJSON bool) error {
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	root, _, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission start: %w", err)
	}
	var status domain.MissionEngineStatus
	err = withMissionLock(deps, root, missionID, func() error {
		s, lockedErr := startLocked(deps, root, missionID)
		if lockedErr != nil {
			return lockedErr
		}
		status = s
		return nil
	})
	if err != nil {
		return err
	}
	return deps.WriteResult(cmd, asJSON, status)
}

// startLocked is RunStart's check-then-create critical section: the existence
// check, engine creation, INITIATIVE consultation, and save. The engine is
// created first so a failed StartMission never leaves an orphaned INITIATIVE
// record. Extracted out of
// RunStart's own lock closure for the same reason as submitLocked.
func startLocked(deps LifecycleDependencies, root, missionID string) (domain.MissionEngineStatus, error) {
	if err := deps.RequireNoExisting(root, missionID); err != nil {
		return domain.MissionEngineStatus{}, err
	}
	engine, status, err := domain.StartMission(domain.MissionStartRequest{MissionID: missionID})
	if err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission start: %w", err)
	}
	if err := runInitiativeStart(deps, root, missionID); err != nil {
		return domain.MissionEngineStatus{}, err
	}
	if err := deps.Save(root, engine.Status()); err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission start: %w", err)
	}
	return status, nil
}

func runInitiativeStart(deps LifecycleDependencies, root, missionID string) error {
	if deps.InitiativeStart == nil {
		return nil
	}
	if err := deps.InitiativeStart(root, missionID); err != nil {
		return fmt.Errorf("mission start: initiative consultation: %w", err)
	}
	return nil
}
