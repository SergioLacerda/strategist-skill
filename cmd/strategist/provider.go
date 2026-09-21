package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	providerpkg "github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Validate and onboard local Strategist providers",
}

type providerOutputOptions struct {
	Format string
}

var providerValidateCmd = &cobra.Command{
	Use:   "validate <source>",
	Short: "Validate a local provider package without changing workspace state",
	Args:  cobra.ExactArgs(1),
}

var providerAddCmd = &cobra.Command{
	Use:   "add <source> --slot <slot>",
	Short: "Stage and bind a validated local provider",
	Args:  cobra.ExactArgs(1),
}

func runProviderValidate(cmd *cobra.Command, args []string, opts providerOutputOptions) error {
	report, err := providerpkg.Validate(args[0], "")
	if printErr := printProviderOutput(cmd, report, opts.Format); printErr != nil {
		return printErr
	}
	if err != nil {
		return fmt.Errorf("provider validate: %w", err)
	}
	return nil
}

func runProviderAdd(cmd *cobra.Command, args []string, root, slot string, opts providerOutputOptions) error {
	if slot == "" {
		return fmt.Errorf("provider add: --slot is required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("provider add: get cwd: %w", err)
	}
	strategistRoot, _, err := cliutil.ResolveStrategistRoot(root, cwd)
	if err != nil {
		return fmt.Errorf("provider add: %w", err)
	}
	result, err := providerpkg.Add(strategistRoot, args[0], slot)
	if printErr := printProviderOutput(cmd, result, opts.Format); printErr != nil {
		return printErr
	}
	if err != nil {
		return fmt.Errorf("provider add: %w", err)
	}
	return nil
}

func printProviderOutput(cmd *cobra.Command, value any, format string) error {
	switch format {
	case cliutil.OutputFormatJSON:
		return printProviderJSON(cmd, value)
	case "yaml":
		return printProviderYAML(cmd, value)
	case cliutil.OutputFormatTable, "":
		return printProviderHuman(cmd, value)
	default:
		return fmt.Errorf("provider: unsupported --format %q", format)
	}
}

func printProviderJSON(cmd *cobra.Command, value any) error {
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(value); err != nil {
		return fmt.Errorf("encode provider output: %w", err)
	}
	return nil
}

func printProviderYAML(cmd *cobra.Command, value any) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal provider output: %w", err)
	}
	if _, err = cmd.OutOrStdout().Write(data); err != nil {
		return fmt.Errorf("write provider output: %w", err)
	}
	return nil
}

func printProviderHuman(cmd *cobra.Command, value any) error {
	out := cmd.OutOrStdout()
	switch report := value.(type) {
	case providerpkg.Report:
		return printProviderReport(out, report)
	case providerpkg.AddResult:
		return writeProviderLine(out, "provider=%s instance=%s slot=%s generation=%d transaction=%s live_invocation=%s\n", report.Report.ProviderID, report.InstanceID, report.Report.RequestedSlot, report.BindingGeneration, report.TransactionState, report.Report.LiveInvocation.Status)
	default:
		return fmt.Errorf("provider: unsupported output type %T", value)
	}
}

func printProviderReport(out io.Writer, report providerpkg.Report) error {
	status := "invalid"
	if report.Validated {
		status = "valid"
	}
	if err := writeProviderLine(out, "provider=%s version=%s status=%s package_digest=%s live_invocation=%s\n", report.ProviderID, report.Version, status, report.PackageDigest, report.LiveInvocation.Status); err != nil {
		return err
	}
	for _, reason := range report.Reasons {
		if err := writeProviderLine(out, "reason=%s detail=%s\n", reason.Code, reason.Detail); err != nil {
			return err
		}
	}
	return nil
}

// writeProviderLine writes one formatted line and reports a failed write, so a
// closed pipe or full disk is not silently swallowed.
func writeProviderLine(out io.Writer, format string, args ...any) error {
	if _, err := fmt.Fprintf(out, format, args...); err != nil {
		return fmt.Errorf("write provider output: %w", err)
	}
	return nil
}

func init() {
	validateOpts := providerOutputOptions{}
	providerValidateCmd.Flags().StringVar(&validateOpts.Format, "format", cliutil.OutputFormatTable, "output format: table, json, or yaml")
	providerValidateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runProviderValidate(cmd, args, validateOpts)
	}

	addOpts := providerOutputOptions{}
	var root, slot string
	providerAddCmd.Flags().StringVar(&root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	providerAddCmd.Flags().StringVar(&slot, "slot", "", "target slot: refinement or execution")
	providerAddCmd.Flags().StringVar(&addOpts.Format, "format", cliutil.OutputFormatTable, "output format: table, json, or yaml")
	providerAddCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runProviderAdd(cmd, args, root, slot, addOpts)
	}

	providerCmd.AddCommand(providerValidateCmd, providerAddCmd)
	rootCmd.AddCommand(providerCmd)
}
