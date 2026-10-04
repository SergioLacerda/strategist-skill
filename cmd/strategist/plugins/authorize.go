package plugins

import (
	"fmt"
	"io"
	"os"

	authorizationapp "github.com/SergioLacerda/strategist-skill/internal/application/authorization"
	"github.com/SergioLacerda/strategist-skill/internal/authorization"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/spf13/cobra"
)

// AuthorizeOptions are the flags of `plugins authorize`.
type AuthorizeOptions struct {
	Root          string
	Target        string
	MissionID     string
	ApprovalGate  string
	ExecutionGate string
	JSON          bool
}

// NewAuthorize creates `plugins authorize`.
func NewAuthorize() *cobra.Command {
	opts := AuthorizeOptions{}
	cmd := &cobra.Command{
		Use:   "authorize",
		Short: "Compose authorization evidence for a governed target",
		Long: `Evaluates runtime readiness, persisted Role→Weapon bindings, the two
independent execution gates, and candidate write-target enforcement. The
command does not invoke external providers or intercept writes performed
outside the CLI. Use --json for stable machine-readable evidence.`,
	}
	cmd.Flags().StringVar(&opts.Root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().StringVar(&opts.Target, "target", "", "candidate write path relative to the project root (required)")
	cmd.Flags().StringVar(&opts.MissionID, "mission", "", "mission identifier to include in the report")
	cmd.Flags().StringVar(&opts.ApprovalGate, "approval-gate", "pending", "Strategist Approval Gate state: pending, accepted, denied, or revision")
	cmd.Flags().StringVar(&opts.ExecutionGate, "execution-gate", "allowed", "local execution gate state: allowed or blocked")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "print the versioned machine-readable authorization report")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunAuthorize(cmd.OutOrStdout(), opts)
	}
	return cmd
}

// RunAuthorize builds and prints the authorization report. The report error is
// wrapped with %w so the root exit-code mapping still sees the
// authorization.ErrDenied/ErrBlocked/ErrStale sentinels.
func RunAuthorize(out io.Writer, opts AuthorizeOptions) error {
	if opts.Target == "" {
		return fmt.Errorf("plugins authorize: --target is required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("plugins authorize: get working directory: %w", err)
	}
	root, _, err := cliutil.ResolveStrategistRoot(opts.Root, cwd)
	if err != nil {
		return fmt.Errorf("plugins authorize: resolve Strategist root: %w", err)
	}
	report, reportErr := authorizationapp.Authorize(authorizationapp.AuthorizationRequest{
		Root: root, Target: opts.Target, MissionID: opts.MissionID,
		ApprovalGate: opts.ApprovalGate, ExecutionGate: opts.ExecutionGate,
	}, func(request authorizationapp.AuthorizationRequest) (authorizationapp.AuthorizationReport, error) {
		authorizationReport, buildErr := authorization.Build(authorization.Request{
			Root: request.Root, Target: request.Target, MissionID: request.MissionID,
			ApprovalGate: request.ApprovalGate, ExecutionGate: request.ExecutionGate,
		})
		return applicationAuthorizationReport(authorizationReport), buildErr
	})
	if err := printAuthorization(out, opts.JSON, authorizationReport(report)); err != nil {
		return err
	}
	if reportErr != nil {
		return fmt.Errorf("plugins authorize: %w", reportErr)
	}
	return nil
}

//nolint:dupl // bidirectional DTO adapters intentionally mirror the report fields.
func applicationAuthorizationReport(report authorization.Report) authorizationapp.AuthorizationReport {
	dimensions := mapAuthorizationDimensions(report.Dimensions, func(dimension authorization.Dimension) authorizationapp.AuthorizationDimension {
		return authorizationapp.AuthorizationDimension{
			Name: dimension.Name, Status: dimension.Status, EvidenceState: dimension.EvidenceState,
			ReasonCode: dimension.ReasonCode, Detail: dimension.Detail, Required: dimension.Required,
		}
	})
	return authorizationapp.AuthorizationReport{
		SchemaVersion: report.SchemaVersion, GeneratedAt: report.GeneratedAt, Root: report.Root,
		Target: report.Target, MissionID: report.MissionID, Role: report.Role,
		Provider: report.Provider, Permission: report.Permission, Decision: report.Decision,
		ReasonCode: report.ReasonCode, ExitClass: report.ExitClass, Dimensions: dimensions,
		Limitations: report.Limitations,
	}
}

//nolint:dupl // bidirectional DTO adapters intentionally mirror the report fields.
func authorizationReport(report authorizationapp.AuthorizationReport) authorization.Report {
	dimensions := mapAuthorizationDimensions(report.Dimensions, func(dimension authorizationapp.AuthorizationDimension) authorization.Dimension {
		return authorization.Dimension{
			Name: dimension.Name, Status: dimension.Status, EvidenceState: dimension.EvidenceState,
			ReasonCode: dimension.ReasonCode, Detail: dimension.Detail, Required: dimension.Required,
		}
	})
	return authorization.Report{
		SchemaVersion: report.SchemaVersion, GeneratedAt: report.GeneratedAt, Root: report.Root,
		Target: report.Target, MissionID: report.MissionID, Role: report.Role,
		Provider: report.Provider, Permission: report.Permission, Decision: report.Decision,
		ReasonCode: report.ReasonCode, ExitClass: report.ExitClass, Dimensions: dimensions,
		Limitations: report.Limitations,
	}
}

func mapAuthorizationDimensions[S any, D any](dimensions []S, convert func(S) D) []D {
	converted := make([]D, 0, len(dimensions))
	for _, dimension := range dimensions {
		converted = append(converted, convert(dimension))
	}
	return converted
}

func printAuthorization(out io.Writer, asJSON bool, report authorization.Report) error {
	if asJSON {
		data, err := report.JSON()
		if err != nil {
			return fmt.Errorf("plugins authorize: encode report: %w", err)
		}
		_, err = fmt.Fprintln(out, string(data))
		return writeErr(err)
	}
	if _, err := fmt.Fprintf(out, "authorization=%s reason=%s target=%s permission=%s\n", report.Decision, report.ReasonCode, report.Target, report.Permission); err != nil {
		return writeErr(err)
	}
	for _, dimension := range report.Dimensions {
		if _, err := fmt.Fprintf(out, "  %s=%s reason=%s evidence=%s\n", dimension.Name, dimension.Status, dimension.ReasonCode, dimension.EvidenceState); err != nil {
			return writeErr(err)
		}
	}
	return nil
}

func writeErr(err error) error {
	if err != nil {
		return fmt.Errorf("plugins authorize: write report: %w", err)
	}
	return nil
}
