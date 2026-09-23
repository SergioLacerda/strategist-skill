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
	"github.com/SergioLacerda/strategist-skill/internal/rolevalidation"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func runCheck(cmd *cobra.Command) error {
	root, err := resolveCheckRoot()
	if err != nil {
		return err
	}
	return withCheckSpan(cmd, root, func(span trace.Span) error {
		return executeCheck(root, span)
	})
}

func withCheckSpan(cmd *cobra.Command, root string, run func(trace.Span) error) (retErr error) {
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
	return run(span)
}

func executeCheck(root string, span trace.Span) error {
	cfg, err := loadActiveConfig(root)
	if err != nil {
		return err
	}
	if checkPrintContentByLang != "" {
		persona := checkPrintContentByLangPersona
		if persona == "" {
			persona = cfg.Mode
		}
		return printContentByLang(root, persona, checkPrintContentByLang)
	}
	language := domain.ParseLanguage(cfg.Language)
	if identityErr := checkIdentityFilesBlockingError(root); identityErr != nil {
		if checkJSON {
			return printPreflightJSONBlocked(root, cfg.Mode, identityErr, language)
		}
		return identityErr
	}

	providers := checkProviders(cfg)
	resolutions, errs := resolveCheckProviders(root, providers)
	diagnostics, weaponBindings := collectCheckDiagnostics(root, cfg, providers, resolutions, language)
	errs = append(errs, diagnostics...)
	decisionReason := checkDecisionReason(providers, errs)
	span.SetAttributes(
		attribute.String(telemetry.AttrPipelineRoute, "main"),
		attribute.String(telemetry.AttrDecisionReason, decisionReason),
	)
	return renderCheck(root, cfg.Mode, providers, resolutions, language, decisionReason, errs, weaponBindings)
}

func resolveCheckRoot() (string, error) {
	root := checkRoot
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("[Strategist] check=blocked reason=cwd_error: %w", err)
		}
		discovered, _, err := cliutil.FindStrategistRoot(cwd)
		if err != nil {
			return "", fmt.Errorf("[Strategist] check=blocked reason=runtime_not_found\n→ Run: strategist install")
		}
		root = discovered
	}
	root, err := absoluteRoot(root)
	if err != nil {
		return "", fmt.Errorf("[Strategist] check=blocked reason=root_unresolvable: %w", err)
	}
	return root, nil
}

func loadActiveConfig(root string) (domain.ActiveConfig, error) {
	activeYAML := filepath.Join(root, "active.yaml")
	raw, err := os.ReadFile(activeYAML) //nolint:gosec // G304: active.yaml path is derived from the selected .strategist root
	if err != nil {
		if os.IsNotExist(err) {
			return domain.ActiveConfig{}, fmt.Errorf("[Strategist] check=blocked reason=active_yaml_not_found\n→ Run: strategist install")
		}
		return domain.ActiveConfig{}, fmt.Errorf("[Strategist] check=blocked reason=active_yaml_read_error: %w", err)
	}
	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return domain.ActiveConfig{}, fmt.Errorf("[Strategist] check=blocked reason=active_yaml_invalid_yaml: %w", err)
	}
	return cfg, nil
}

func resolveCheckProviders(root string, providers map[string]string) (map[string]slotResolution, []string) {
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
	errs = append(errs, checkPluginLockParity(root, providers)...)
	return resolutions, errs
}

func collectCheckDiagnostics(root string, cfg domain.ActiveConfig, providers map[string]string, resolutions map[string]slotResolution, language *domain.PreflightLanguage) ([]string, []weaponBinding) {
	var errs []string
	for _, failure := range rolevalidation.ValidateRuntimeBindings(root, cfg) {
		errs = append(errs, failure.Error())
	}
	errs = append(errs, blockedReadinessErrorsForSlots(resolutions, []string{"discovery", "refinement", "execution"})...)
	errs = append(errs, validateActivePersona(root, cfg.Mode)...)
	weaponBindings, weaponErr := verifyEmbeddedWeaponBindings(root)
	if weaponErr != nil {
		errs = append(errs, weaponErr.Error())
	}
	errs = append(errs, weaponBindingErrors(weaponBindings)...)
	errs = append(errs, checkPluginLockParity(root, providers)...)
	errs = append(errs, validateRuntimeDefaultParity(root)...)
	if err := emitF3ConflictAttributionSignals(root, cfg.BasePath, time.Now()); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ f3_conflict_signal: %v\n", err)
	}
	if checkStrict {
		errs = append(errs, runStrictChecks(root)...)
	}
	if checkConfirmChatLanguage != "" && language != nil && language.Chat != "" && checkConfirmChatLanguage != language.Chat {
		errs = append(errs, fmt.Sprintf("chat_language_mismatch: confirmed=%s configured=%s", checkConfirmChatLanguage, language.Chat))
	}
	return errs, weaponBindings
}

func renderCheck(root, mode string, providers map[string]string, resolutions map[string]slotResolution, language *domain.PreflightLanguage, decisionReason string, errs []string, weaponBindings []weaponBinding) error {
	if checkSimulate {
		return printSimulateReport(root, providers, resolutions, mode, decisionReason, errs)
	}
	if checkJSON {
		return printPreflightJSON(root, mode, providers, resolutions, errs, language)
	}
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  ✗ %s\n", e)
		}
		return fmt.Errorf("[Strategist] check=failed errors=%d root=%s", len(errs), root)
	}
	return printCheckSuccess(root, providers, resolutions, mode, weaponBindings)
}
