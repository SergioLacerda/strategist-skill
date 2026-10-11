package mission

import (
	"errors"
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/hostusage"
	"github.com/spf13/cobra"
)

// ObserveUsageOptions carries the `mission observe-usage` flag values.
type ObserveUsageOptions struct {
	Root, MissionID, Host, Phase, Role string
	InvocationID, SessionID            string
	Since, Until                       string
	Transcripts                        []string
	IncludeSubagents, DryRun           bool
}

// lagNotice is how recent a window end may be before the host file is likely
// to lag the live conversation.
const lagNotice = 30 * time.Second

// NewObserveUsage builds `mission observe-usage`: record the usage a coding
// host already wrote in its own session file for one time window.
func NewObserveUsage(deps ReportUsageDependencies) *cobra.Command {
	opts := ObserveUsageOptions{}
	cmd := &cobra.Command{
		Use:   "observe-usage",
		Short: "Record the usage a host wrote in its own session file for one time window (opt-in, numbers only)",
		Long: `strategist mission observe-usage reads a session file that Claude Code or Codex
already wrote, for the file(s) you name with --transcript, and records the
token counters the host itself logged inside the window [--since, --until).

It is opt-in per run and never scans a home directory. The reader keeps only
ids, timestamps, model names and numeric counters; conversation content is
read past and never stored, printed or logged. Quality is host_measured, never
provider_measured. Claude Code responses are deduplicated by message id and
sub-agent files next to the transcript are included; Codex uses the delta of
its cumulative counter. Input includes cached input; the cache part is kept
in its own field.

Attribution is by time window only. The same host session and window is
counted once even if re-run; a re-run that sees different counts is a
conflict, because the host file can lag the live conversation. A window with
no host calls records nothing (never zero). The formats are not contracts:
an unrecognized shape fails closed.`,
	}
	f := cmd.Flags()
	f.StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.MissionID, "mission-id", "", "mission_id to record usage against (required)")
	f.StringVar(&opts.Host, "host", "", "host that wrote the file (required): claude or codex")
	f.StringSliceVar(&opts.Transcripts, "transcript", nil, "session file to read (required, repeatable)")
	f.BoolVar(&opts.IncludeSubagents, "include-subagents", true, "claude: also read <transcript>/subagents/*.jsonl")
	f.StringVar(&opts.Since, "since", "", "window start, RFC 3339, inclusive (required)")
	f.StringVar(&opts.Until, "until", "", "window end, RFC 3339, exclusive (required)")
	f.StringVar(&opts.Phase, "phase", "", "phase the window belongs to (required)")
	f.StringVar(&opts.Role, "role", "", "role that consumed the tokens (required)")
	f.StringVar(&opts.InvocationID, "invocation-id", "", "the 'mission invoke' request id the window belongs to, when known")
	f.StringVar(&opts.SessionID, "session-id", "", "host session id when the file does not carry one")
	f.BoolVar(&opts.DryRun, "dry-run", false, "print what would be recorded without writing the ledger")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunObserveUsage(cmd, deps, opts) }
	return cmd
}

// RunObserveUsage observes one window and records it.
func RunObserveUsage(cmd *cobra.Command, deps ReportUsageDependencies, opts ObserveUsageOptions) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	window, err := observeWindow(opts, time.Now())
	if err != nil {
		return fmt.Errorf("mission observe-usage: %w", err)
	}
	obs, notes, err := observeHost(opts, window)
	if err != nil {
		return fmt.Errorf("mission observe-usage: %w", err)
	}
	strategistRoot, basePath, err := deps.ResolveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("mission observe-usage: %w", err)
	}
	in := hostInput(opts, obs)
	if opts.DryRun {
		return writeObservation(cmd, obs, notes, "dry-run (nothing written)")
	}
	res, err := application.RecordHostObservedUsage(strategistRoot, basePath, in, time.Now(), application.MissionUsagePorts{
		MissionKnown: deps.MissionKnown, Append: usageAppender(deps.TelemetrySink),
	})
	if err != nil {
		return fmt.Errorf("mission observe-usage: %w", err)
	}
	status := "recorded"
	if !res.Appended {
		status = "duplicate (this host session and window were already recorded)"
	}
	return writeObservation(cmd, obs, notes, status)
}

func observeWindow(opts ObserveUsageOptions, now time.Time) (hostusage.Window, error) {
	since, err := time.Parse(time.RFC3339, opts.Since)
	if err != nil {
		return hostusage.Window{}, fmt.Errorf("--since must be RFC 3339: %w", err)
	}
	until, err := time.Parse(time.RFC3339, opts.Until)
	if err != nil {
		return hostusage.Window{}, fmt.Errorf("--until must be RFC 3339: %w", err)
	}
	if until.After(now) {
		return hostusage.Window{}, errors.New("--until is in the future: the host has not written that window yet")
	}
	w := hostusage.Window{Start: since, End: until}
	if err := w.Validate(); err != nil {
		return hostusage.Window{}, fmt.Errorf("window: %w", err)
	}
	return w, nil
}

// observeHost reads every named file and returns the window observation and
// the coverage notes to print.
func observeHost(opts ObserveUsageOptions, w hostusage.Window) (hostusage.Observation, []string, error) {
	if len(opts.Transcripts) == 0 {
		return hostusage.Observation{}, nil, errors.New("--transcript is required")
	}
	switch opts.Host {
	case hostusage.HostClaude:
		return observeClaudeFiles(opts, w)
	case hostusage.HostCodex:
		return observeCodexFile(opts, w)
	}
	return hostusage.Observation{}, nil, fmt.Errorf("--host %q must be claude or codex", opts.Host)
}

func hostInput(opts ObserveUsageOptions, obs hostusage.Observation) application.HostObservedUsageInput {
	session := obs.SessionID
	if opts.SessionID != "" {
		session = opts.SessionID
	}
	return application.HostObservedUsageInput{
		MissionID: opts.MissionID, Phase: opts.Phase, Role: opts.Role, InvocationID: opts.InvocationID,
		Host: obs.Host, SessionID: session, Provider: obs.Provider, Models: obs.Models,
		WindowStart: obs.WindowStart, WindowEnd: obs.WindowEnd, Calls: obs.Calls,
		Input: obs.Input, Output: obs.Output, Cache: obs.Cache, Reasoning: obs.Reasoning,
	}
}

func writeObservation(cmd *cobra.Command, obs hostusage.Observation, notes []string, status string) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "host=%s calls=%d tokens_in=%s tokens_out=%s tokens_cache=%s quality=host_measured window=[%s,%s) %s\n",
		obs.Host, obs.Calls, countText(obs.Input), countText(obs.Output), countText(obs.Cache),
		obs.WindowStart.UTC().Format(time.RFC3339), obs.WindowEnd.UTC().Format(time.RFC3339), status); err != nil {
		return fmt.Errorf("mission observe-usage: write output: %w", err)
	}
	for _, n := range notes {
		if _, err := fmt.Fprintf(out, "note: %s\n", n); err != nil {
			return fmt.Errorf("mission observe-usage: write output: %w", err)
		}
	}
	if time.Since(obs.WindowEnd) < lagNotice {
		if _, err := fmt.Fprintln(out, "note: the window ends within the last seconds; the host file may lag, so a later re-run could conflict"); err != nil {
			return fmt.Errorf("mission observe-usage: write output: %w", err)
		}
	}
	return nil
}
