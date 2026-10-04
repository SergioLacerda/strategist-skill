package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/setup"
	"github.com/spf13/cobra"
)

type integrationsOptions struct {
	Root          string
	JSON          bool
	Endpoint      string
	CredentialRef string
	Model         string
}

var integrationsCmd = &cobra.Command{
	Use:   "integrations",
	Short: "Inspect and change the optional external provider integration",
	Long: `Manage the optional provider integration (JEV) that pre-checks handoff conformance.
Strategist works unchanged without it: a provider that is disabled, unreachable or not
confident enough leaves every handoff on the main path. The operator decision lives in
.strategist/integrations.yaml; the credential is only a reference (env: or dotenv:) and is
never stored, printed or sent anywhere but the configured endpoint.`,
}

func integrationsRoot(opts integrationsOptions) (string, error) {
	root, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return "", fmt.Errorf("integrations: %w", err)
	}
	return root, nil
}

func loadIntegrations(root string) (config.File, bool, error) {
	file, found, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		return config.File{}, false, fmt.Errorf("integrations: %w", err)
	}
	return file, found, nil
}

// decide runs the shared planner for a selection and persists it.
func decide(root string, choice setup.Choice, opts integrationsOptions) (setup.Plan, error) {
	file, found, err := loadIntegrations(root)
	if err != nil {
		return setup.Plan{}, err
	}
	overrides := setup.Overrides{Endpoint: opts.Endpoint, CredentialRef: opts.CredentialRef, Model: opts.Model}
	ref := firstNonEmpty(opts.CredentialRef, file.Providers[setup.ProviderName].CredentialRef, setup.DefaultCredentialRef)
	plan, err := setup.Decide(setup.Input{
		Existing: file, Found: found, Choice: choice, Overrides: overrides,
		CredentialResolves: setup.Resolves(ref, filepath.Dir(root)),
	})
	if err != nil {
		return setup.Plan{}, fmt.Errorf("integrations: %w", err)
	}
	if err := setup.Apply(filepath.Join(root, config.FileName), plan); err != nil {
		return setup.Plan{}, fmt.Errorf("integrations: %w", err)
	}
	return plan, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func runIntegrationsEnable(cmd *cobra.Command, opts integrationsOptions) error {
	root, err := integrationsRoot(opts)
	if err != nil {
		return err
	}
	plan, err := decide(root, setup.ChoiceEnable, opts)
	if err != nil {
		return err
	}
	provider := plan.File.Providers[setup.ProviderName]
	credential := "resolves"
	if plan.Pending {
		credential = "pending (not resolvable yet; the provider falls back to main until it is)"
	}
	return printLines(cmd, "integration: enabled",
		"endpoint: "+plan.Endpoint, "model: "+provider.Model, "credential: "+credential,
		"data: "+strings.Join(plan.DataCategories, ", "))
}

func runIntegrationsDisable(cmd *cobra.Command, opts integrationsOptions) error {
	root, err := integrationsRoot(opts)
	if err != nil {
		return err
	}
	if _, err := decide(root, setup.ChoiceDisable, opts); err != nil {
		return err
	}
	return printLines(cmd, "integration: disabled")
}

func printLines(cmd *cobra.Command, lines ...string) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(lines, "\n")); err != nil {
		return fmt.Errorf("print integrations output: %w", err)
	}
	return nil
}

func init() {
	for _, spec := range []struct {
		use, short string
		run        func(*cobra.Command, integrationsOptions) error
		flags      func(*cobra.Command, *integrationsOptions)
	}{
		{"doctor", "Report declared, bound, enabled, compatible and available state without calling the provider", runIntegrationsDoctor, jsonFlag},
		{"probe", "Send one synthetic, non-mission message to record availability", runIntegrationsProbe, nil},
		{"enable", "Record the decision to use the provider (documented defaults, credential stays a reference)", runIntegrationsEnable, enableFlags},
		{"disable", "Record the decision not to use the provider; recorded outcomes and settings are kept", runIntegrationsDisable, nil},
	} {
		integrationsCmd.AddCommand(integrationsSubcommand(spec.use, spec.short, spec.run, spec.flags))
	}
}

func integrationsSubcommand(use, short string, run func(*cobra.Command, integrationsOptions) error, flags func(*cobra.Command, *integrationsOptions)) *cobra.Command {
	opts := &integrationsOptions{}
	cmd := &cobra.Command{Use: use, Short: short, SilenceUsage: true, Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return run(cmd, *opts) }}
	cmd.Flags().StringVar(&opts.Root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	if flags != nil {
		flags(cmd, opts)
	}
	return cmd
}

func jsonFlag(cmd *cobra.Command, opts *integrationsOptions) {
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "emit machine-readable JSON")
}

func enableFlags(cmd *cobra.Command, opts *integrationsOptions) {
	cmd.Flags().StringVar(&opts.Endpoint, "endpoint", "", "provider HTTPS endpoint (default: the documented JEV endpoint)")
	cmd.Flags().StringVar(&opts.CredentialRef, "credential-ref", "", "credential reference, env:NAME or dotenv:<path>#NAME (default: dotenv:.env#TYPESAFE_API_KEY)")
	cmd.Flags().StringVar(&opts.Model, "model", "", "pinned provider model (default: the documented resolved model)")
}
