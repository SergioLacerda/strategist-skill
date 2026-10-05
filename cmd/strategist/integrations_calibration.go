package main

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/spf13/cobra"
)

// calibrationResolver loads the pinned calibration from the DOJO storage domain
// of the workspace the integration file belongs to.
func calibrationResolver(root string) handoffconsumer.CalibrationResolver {
	return func(id string) (calibration.Record, error) {
		_, basePath, err := cliutil.ResolveActiveBasePath(root)
		if err != nil {
			return calibration.Record{}, fmt.Errorf("calibration: %w", err)
		}
		return dojo.LoadCalibration(basePath, id)
	}
}

type calibrationOptions struct {
	integrationsOptions
	confirm bool
	maxAge  int
}

var integrationsCalibrationCmd = &cobra.Command{
	Use:   "calibration",
	Short: "List, show, pin and unpin the calibration the pre-check threshold follows",
	Long: `The approval threshold stays operator configuration. A calibration published by a DOJO case
(strategist dojo calibrate) is pinned here by id, never "latest": the pre-check uses its threshold only
while it is valid for the resolved model and the question profile and is not too old, and otherwise
uses the configured threshold and reports "não calibrado". Nothing pins automatically.`,
}

func init() {
	opts := &calibrationOptions{}
	list := &cobra.Command{Use: "list", Short: "List the published calibrations", Args: cobra.NoArgs, SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error { return runCalibrationList(cmd, opts.integrationsOptions) }}
	show := &cobra.Command{Use: "show [id]", Short: "Show a calibration with its numbers and risk policy (default: the pinned one)", Args: cobra.MaximumNArgs(1), SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCalibrationShow(cmd, opts.integrationsOptions, firstArg(args))
		}}
	pin := &cobra.Command{Use: "pin <id>", Short: "Pin a calibration after showing its numbers; --confirm writes the pin", Args: cobra.ExactArgs(1), SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error { return runCalibrationPin(cmd, *opts, args[0]) }}
	unpin := &cobra.Command{Use: "unpin", Short: "Remove the pin; the configured threshold applies", Args: cobra.NoArgs, SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error { return runCalibrationUnpin(cmd, opts.integrationsOptions) }}
	for _, c := range []*cobra.Command{list, show, pin, unpin} {
		c.Flags().StringVar(&opts.Root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
		integrationsCalibrationCmd.AddCommand(c)
	}
	pin.Flags().BoolVar(&opts.confirm, "confirm", false, "write the pin after reviewing the numbers")
	pin.Flags().IntVar(&opts.maxAge, "max-age-days", 0, "age limit of the pin in days (0 keeps the default of 90)")
	integrationsCmd.AddCommand(integrationsCalibrationCmd)
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func calibrationBase(opts integrationsOptions) (root, basePath string, err error) {
	root, basePath, err = cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return "", "", fmt.Errorf("integrations calibration: %w", err)
	}
	return root, basePath, nil
}

// pinTarget is what a pin acts on: the record, the integration file and its provider.
type pinTarget struct {
	root     string
	record   calibration.Record
	file     config.File
	provider config.Provider
}

func loadPinTarget(opts integrationsOptions, id string) (pinTarget, error) {
	root, basePath, err := calibrationBase(opts)
	if err != nil {
		return pinTarget{}, err
	}
	record, err := dojo.LoadCalibration(basePath, id)
	if err != nil {
		return pinTarget{}, fmt.Errorf("integrations calibration: %w", err)
	}
	file, found, err := loadIntegrations(root)
	if err != nil {
		return pinTarget{}, err
	}
	provider, declared := file.Providers[integrationProvider]
	if !found || !declared {
		return pinTarget{}, fmt.Errorf("integrations calibration: no %s integration is declared; run `strategist integrations enable` first", integrationProvider)
	}
	return pinTarget{root: root, record: record, file: file, provider: provider}, nil
}

func runCalibrationPin(cmd *cobra.Command, opts calibrationOptions, id string) error {
	target, err := loadPinTarget(opts.integrationsOptions, id)
	if err != nil {
		return err
	}
	if err := pinnable(target.record, target.provider); err != nil {
		return err
	}
	if !opts.confirm {
		return printLines(cmd, append(calibrationLines(target.record), "", "not pinned: review the numbers and the risk policy above, then run again with --confirm")...)
	}
	target.provider.CalibrationID, target.provider.CalibrationMaxAgeDays = target.record.ID, opts.maxAge
	target.file.Providers[integrationProvider] = target.provider
	if err := config.Save(filepath.Join(target.root, config.FileName), target.file); err != nil {
		return fmt.Errorf("integrations calibration: %w", err)
	}
	return printLines(cmd, "pinned "+target.record.ID, fmt.Sprintf("threshold %.2f applies while the model, the question profile and the age stay valid; otherwise the configured %.2f", *target.record.Threshold, target.provider.Threshold()))
}

// pinnable refuses a record the handoff could not use: one without a
// recommendation, for another model than the pinned one, or for an old profile.
func pinnable(record calibration.Record, provider config.Provider) error {
	facts := handoffconsumer.FactsOf(record)
	switch {
	case record.Status != calibration.StatusRecommended:
		return fmt.Errorf("integrations calibration: %s is %s and carries no threshold; it cannot be pinned", record.ID, record.Status)
	case record.Model != provider.Model && !provider.AllowModelAlias:
		return fmt.Errorf("integrations calibration: %s is for model %s but the pinned model is %s", record.ID, record.Model, provider.Model)
	case facts.Profile != calibration.ResolveProfile(provider.Profile) || !profileCurrent(record, provider.Profile):
		return fmt.Errorf("integrations calibration: %s was taken with another question profile (or version) than the provider's configured one; run `strategist dojo calibrate` again, or change the provider profile", record.ID)
	}
	return nil
}

func profileCurrent(record calibration.Record, profile string) bool {
	id := calibration.ResolveProfile(profile)
	return record.ProfileVersion == handoffconsumer.ProfileVersionOf(id, calibration.BoundaryPackage) || record.ProfileVersion == handoffconsumer.ProfileVersionOf(id, calibration.BoundaryRanger)
}

func runCalibrationUnpin(cmd *cobra.Command, opts integrationsOptions) error {
	root, err := integrationsRoot(opts)
	if err != nil {
		return err
	}
	file, found, err := loadIntegrations(root)
	if err != nil {
		return err
	}
	provider, declared := file.Providers[integrationProvider]
	if !found || !declared || provider.CalibrationID == "" {
		return printLines(cmd, "no calibration is pinned")
	}
	provider.CalibrationID, provider.CalibrationMaxAgeDays = "", 0
	file.Providers[integrationProvider] = provider
	if err := config.Save(filepath.Join(root, config.FileName), file); err != nil {
		return fmt.Errorf("integrations calibration: %w", err)
	}
	return printLines(cmd, "unpinned: the configured threshold applies (não calibrado)")
}
