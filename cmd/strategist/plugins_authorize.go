package main

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/authorization"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/spf13/cobra"
)

type pluginsAuthorizeOptions struct {
	Root          string
	Target        string
	MissionID     string
	ApprovalGate  string
	ExecutionGate string
	JSON          bool
}

var pluginsAuthorizeCmd = &cobra.Command{
	Use:   "authorize",
	Short: "Compose authorization evidence for a governed target",
	Long: `Evaluates runtime readiness, persisted Role→Weapon bindings, the two
independent execution gates, and candidate write-target enforcement. The
command does not invoke external providers or intercept writes performed
outside the CLI. Use --json for stable machine-readable evidence.`,
}

func runPluginsAuthorize(opts pluginsAuthorizeOptions) error {
	if opts.Target == "" {
		return fmt.Errorf("plugins authorize: --target is required")
	}
	root, err := resolvePluginsAuthorizeRoot(opts.Root)
	if err != nil {
		return fmt.Errorf("plugins authorize: %w", err)
	}
	report, reportErr := authorization.Build(authorization.Request{
		Root: root, Target: opts.Target, MissionID: opts.MissionID,
		ApprovalGate: opts.ApprovalGate, ExecutionGate: opts.ExecutionGate,
	})
	if err := printPluginsAuthorization(opts, report); err != nil {
		return err
	}
	if reportErr != nil {
		return fmt.Errorf("plugins authorize: %w", reportErr)
	}
	return nil
}

func resolvePluginsAuthorizeRoot(configuredRoot string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	root, _, err := cliutil.ResolveStrategistRoot(configuredRoot, cwd)
	if err != nil {
		return "", fmt.Errorf("resolve Strategist root: %w", err)
	}
	return root, nil
}

func printPluginsAuthorization(opts pluginsAuthorizeOptions, report authorization.Report) error {
	if opts.JSON {
		data, marshalErr := report.JSON()
		if marshalErr != nil {
			return fmt.Errorf("plugins authorize: encode report: %w", marshalErr)
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("authorization=%s reason=%s target=%s permission=%s\n", report.Decision, report.ReasonCode, report.Target, report.Permission)
		for _, dimension := range report.Dimensions {
			fmt.Printf("  %s=%s reason=%s evidence=%s\n", dimension.Name, dimension.Status, dimension.ReasonCode, dimension.EvidenceState)
		}
	}
	return nil
}

func init() {
	opts := pluginsAuthorizeOptions{}
	pluginsAuthorizeCmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	pluginsAuthorizeCmd.Flags().StringVar(&opts.Target, "target", "", "candidate write path relative to the project root (required)")
	pluginsAuthorizeCmd.Flags().StringVar(&opts.MissionID, "mission", "", "mission identifier to include in the report")
	pluginsAuthorizeCmd.Flags().StringVar(&opts.ApprovalGate, "approval-gate", "pending", "Strategist Approval Gate state: pending, accepted, denied, or revision")
	pluginsAuthorizeCmd.Flags().StringVar(&opts.ExecutionGate, "execution-gate", "allowed", "local execution gate state: allowed or blocked")
	pluginsAuthorizeCmd.Flags().BoolVar(&opts.JSON, "json", false, "print the versioned machine-readable authorization report")
	pluginsAuthorizeCmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runPluginsAuthorize(opts)
	}
	pluginsCmd.AddCommand(pluginsAuthorizeCmd)
}
