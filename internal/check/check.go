package check

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	checkRoot                      string
	checkStrict                    bool
	checkSimulate                  bool
	checkPrintContentByLang        string
	checkPrintContentByLangPersona string
	readGitConflictedPaths         = readGitConflictedPathsFromWorktree
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Pre-mission slot validation",
	Long: `Validate that slot providers declared in active.yaml are installed and
satisfy their risk_score contracts.

Checks performed:
  - active.yaml is present and parseable
  - For each slot (discovery, refinement, execution):
      • skills/<provider>/skill.yaml exists (skill provider), OR
        roles/<provider>.yaml exists with matching slot field (native role)
      • skill providers must declare the correct risk_score for the slot contract:
        discovery/refinement → write_analysis, execution → controlled
      • native roles are accepted by slot field match; no risk_score check`,
	RunE: func(cmd *cobra.Command, _ []string) (retErr error) {
		root := checkRoot
		if root == "" {
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				return fmt.Errorf("[Strategist] check=blocked reason=cwd_error: %w", cwdErr)
			}
			discovered, _, discErr := cliutil.FindStrategistRoot(cwd)
			if discErr != nil {
				return fmt.Errorf("[Strategist] check=blocked reason=runtime_not_found\n→ Run: strategist install")
			}
			root = discovered
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		_, span := telemetry.Tracer().Start(ctx, "strategist.check",
			trace.WithAttributes(
				attribute.String(telemetry.AttrComponent, "check"),
				attribute.String(telemetry.AttrTarget, telemetry.SanitizePath(root)),
			),
		)
		defer func() {
			if retErr != nil {
				span.RecordError(retErr)
				span.SetStatus(codes.Error, retErr.Error())
			}
			span.End()
		}()

		activeYAML := filepath.Join(root, "active.yaml")
		raw, err := os.ReadFile(activeYAML) //nolint:gosec // G304: active.yaml path is derived from the selected .strategist root
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("[Strategist] check=blocked reason=active_yaml_not_found\n→ Run: strategist install")
			}
			return fmt.Errorf("[Strategist] check=blocked reason=active_yaml_read_error: %w", err)
		}

		var cfg domain.ActiveConfig
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("[Strategist] check=blocked reason=active_yaml_invalid_yaml: %w", err)
		}

		if checkPrintContentByLang != "" {
			persona := checkPrintContentByLangPersona
			if persona == "" {
				persona = cfg.Mode
			}
			return printContentByLang(root, persona, checkPrintContentByLang)
		}

		if identityErr := checkIdentityFilesBlockingError(root); identityErr != nil {
			return identityErr
		}

		providers := map[string]string{
			"discovery":  cfg.Slots["discovery"],
			"refinement": cfg.Slots["refinement"],
			"execution":  cfg.Slots["execution"],
		}

		resolutions := map[string]slotResolution{}
		var errs []string
		for _, slot := range []string{"discovery", "refinement", "execution"} {
			provider := providers[slot]
			if provider == "" {
				errs = append(errs, fmt.Sprintf("slot %s: no provider configured in active.yaml", slot))
				continue
			}
			res, errMsg := resolveSlotProvider(root, slot, provider)
			if errMsg != "" {
				errs = append(errs, errMsg)
				continue
			}
			resolutions[slot] = res
		}

		// Validate active persona.
		if cfg.Mode == "" {
			errs = append(errs, "active.yaml: mode is empty — must be epic or pragmatic")
		} else {
			personaPath := filepath.Join(root, "personas", cfg.Mode+".yaml")
			personaRaw, personaErr := os.ReadFile(personaPath) //nolint:gosec // G304: persona path is derived from active runtime mode
			if personaErr != nil {
				errs = append(errs, fmt.Sprintf("persona: mode=%q file missing (%s)", cfg.Mode, personaPath))
			} else {
				var persona domain.PersonaConfig
				if yamlErr := yaml.Unmarshal(personaRaw, &persona); yamlErr != nil {
					errs = append(errs, fmt.Sprintf("persona: mode=%q invalid yaml: %v", cfg.Mode, yamlErr))
				} else if rtErr := persona.ValidateForRuntime(); rtErr != nil {
					errs = append(errs, fmt.Sprintf("persona: mode=%q %v", cfg.Mode, rtErr))
				}
			}
		}

		errs = append(errs, validateRuntimeDefaultParity(root)...)
		emitErr := emitF3ConflictAttributionSignals(root, cfg.BasePath, time.Now())
		if emitErr != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ f3_conflict_signal: %v\n", emitErr)
		}

		if checkStrict {
			errs = append(errs, runStrictChecks(root)...)
		}

		decisionReason := "all_slots_ready"
		for _, slot := range []string{"discovery", "refinement", "execution"} {
			if providers[slot] == "" {
				decisionReason = "slot_provider_missing:" + slot
				break
			}
		}
		if decisionReason == "all_slots_ready" && len(errs) > 0 {
			decisionReason = "validation_failed"
		}
		span.SetAttributes(
			attribute.String(telemetry.AttrPipelineRoute, "main"),
			attribute.String(telemetry.AttrDecisionReason, decisionReason),
		)

		if checkSimulate {
			return printSimulateReport(root, providers, resolutions, cfg.Mode, decisionReason, errs)
		}

		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  ✗ %s\n", e)
			}
			return fmt.Errorf("[Strategist] check=failed errors=%d root=%s", len(errs), root)
		}

		return printCheckSuccess(root, providers, resolutions, cfg.Mode)
	},
}

func init() {
	checkCmd.Flags().StringVar(&checkRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	checkCmd.Flags().BoolVar(&checkStrict, "strict", false, "additionally require compiled artifacts to exist and match the recorded manifest hashes")
	checkCmd.Flags().BoolVar(&checkSimulate, "simulate", false, "print a readiness report (per-slot/persona status) instead of the pass/fail banner; never invokes providers or mutates state")
	checkCmd.Flags().StringVar(&checkPrintContentByLang, "print-content-by-lang", "", "print personas.<persona>.content_by_lang.<lang> and phase_announcements.<lang> from the compiled artifact as JSON and exit; use this instead of reading persona YAML directly to resolve non-English chat templates and mission narration lines")
	checkCmd.Flags().StringVar(&checkPrintContentByLangPersona, "persona", "", "persona/mode to read with --print-content-by-lang (default: active.yaml's mode)")
}

// Register attaches this package's top-level commands (check, check-stale)
// onto root. Called once from cmd/strategist's own init(), mirroring how
// every other feature area self-registers onto rootCmd — see
// cmd/strategist/check_register.go and internal/treasurecli's own Register.
func Register(root *cobra.Command) {
	root.AddCommand(checkCmd)
	root.AddCommand(checkStaleCmd)
}
