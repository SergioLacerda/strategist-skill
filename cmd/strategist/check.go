package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
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

// slotContract maps slot names to their required risk_score contract.
var slotContract = map[string]string{
	"discovery":  "write_analysis",
	"refinement": "write_analysis",
	"execution":  "controlled",
}

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
			discovered, _, discErr := findStrategistRoot(cwd)
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
		raw, err := os.ReadFile(activeYAML)
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

		providers := map[string]string{
			"discovery":  cfg.Slots["discovery"],
			"refinement": cfg.Slots["refinement"],
			"execution":  cfg.Slots["execution"],
		}

		var errs []string
		for _, slot := range []string{"discovery", "refinement", "execution"} {
			provider := providers[slot]
			if provider == "" {
				errs = append(errs, fmt.Sprintf("slot %s: no provider configured in active.yaml", slot))
				continue
			}

			skillPath := filepath.Join(root, "skills", provider, "skill.yaml")
			skillRaw, readErr := os.ReadFile(skillPath)
			if readErr != nil {
				if os.IsNotExist(readErr) {
					// Fallback: accept native roles declared in roles/<provider>.yaml.
					rolePath := filepath.Join(root, "roles", provider+".yaml")
					roleRaw, roleErr := os.ReadFile(rolePath)
					if roleErr == nil {
						var roleDef domain.RoleConfig
						if yamlErr := yaml.Unmarshal(roleRaw, &roleDef); yamlErr == nil {
							if valErr := roleDef.Validate(); valErr != nil {
								errs = append(errs, fmt.Sprintf("slot %s: role %q invalid: %v", slot, provider, valErr))
								continue
							}
							if roleDef.Slot == slot {
								continue // valid native role for this slot
							}
							errs = append(errs, fmt.Sprintf("slot %s: role %q declares slot=%q (mismatch)", slot, provider, roleDef.Slot))
							continue
						}
					}
					errs = append(errs, fmt.Sprintf("slot %s: provider %q not installed (missing %s)", slot, provider, skillPath))
				} else {
					errs = append(errs, fmt.Sprintf("slot %s: read %s: %v", slot, skillPath, readErr))
				}
				continue
			}

			var skillDef struct {
				RiskScore string `yaml:"risk_score"`
			}
			if yamlErr := yaml.Unmarshal(skillRaw, &skillDef); yamlErr != nil {
				errs = append(errs, fmt.Sprintf("slot %s: provider %q skill.yaml invalid: %v", slot, provider, yamlErr))
				continue
			}

			required := slotContract[slot]
			if skillDef.RiskScore != required {
				errs = append(errs, fmt.Sprintf("slot %s: provider %q has risk_score=%q but slot requires %q — preflight will block", slot, provider, skillDef.RiskScore, required))
			}
		}

		// Validate active persona.
		if cfg.Mode == "" {
			errs = append(errs, "active.yaml: mode is empty — must be epic or pragmatic")
		} else {
			personaPath := filepath.Join(root, "personas", cfg.Mode+".yaml")
			personaRaw, personaErr := os.ReadFile(personaPath)
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
			return printSimulateReport(root, providers, cfg.Mode, decisionReason, errs)
		}

		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  ✗ %s\n", e)
			}
			return fmt.Errorf("[Strategist] check=failed errors=%d root=%s", len(errs), root)
		}

		printStatusBanner("check")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(w, "STATUS\t"); err != nil {
			return fmt.Errorf("check: write status header: %w", err)
		}
		if _, err := fmt.Fprintf(w, "  ok\troot=%s\n", root); err != nil {
			return fmt.Errorf("check: write status row: %w", err)
		}
		if _, err := fmt.Fprintln(w, "\t"); err != nil {
			return fmt.Errorf("check: write separator: %w", err)
		}
		if _, err := fmt.Fprintln(w, "SLOTS\t"); err != nil {
			return fmt.Errorf("check: write slots header: %w", err)
		}
		for _, slot := range []string{"discovery", "refinement", "execution"} {
			if _, err := fmt.Fprintf(w, "  %-12s\t%s\n", slot, providers[slot]); err != nil {
				return fmt.Errorf("check: write slot row: %w", err)
			}
		}
		if _, err := fmt.Fprintln(w, "\t"); err != nil {
			return fmt.Errorf("check: write separator: %w", err)
		}
		if _, err := fmt.Fprintln(w, "PERSONA\t"); err != nil {
			return fmt.Errorf("check: write persona header: %w", err)
		}
		if _, err := fmt.Fprintf(w, "  mode\t%s\n", cfg.Mode); err != nil {
			return fmt.Errorf("check: write persona row: %w", err)
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("check: flush output: %w", err)
		}
		return nil
	},
}

func emitF3ConflictAttributionSignals(strategistRoot, basePath string, now time.Time) error {
	worktreeRoot := filepath.Dir(strategistRoot)
	conflicted, err := readGitConflictedPaths(worktreeRoot)
	if err != nil {
		return err
	}
	records, err := telemetry.ReadRecentSniperMaterializations(
		telemetry.SniperMaterializationHistoryPath(strategistRoot),
		now,
		telemetry.SniperMaterializationWindow,
	)
	if err != nil {
		return fmt.Errorf("read recent sniper materializations: %w", err)
	}
	for _, signal := range telemetry.SniperConflictSignals(basePath, conflicted, records) {
		telemetry.EmitSniperConflictSignal(signal)
	}
	return nil
}

func readGitConflictedPathsFromWorktree(worktreeRoot string) ([]string, error) {
	//nolint:gosec // G204: read-only fixed git subcommand; worktreeRoot is the discovered workspace root.
	cmd := exec.Command("git", "-C", worktreeRoot, "diff", "--name-only", "--diff-filter=U")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "not a git repository") {
			return nil, nil
		}
		return nil, fmt.Errorf("read git conflicted paths: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return parseGitPathLines(string(out)), nil
}

func parseGitPathLines(out string) []string {
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(path))
		if clean != "." {
			paths = append(paths, clean)
		}
	}
	return paths
}

// runStrictChecks composes the additional checks --strict adds on top of the
// base check: required compiled artifacts must exist, and their recorded
// manifest hashes must match the artifacts on disk.
func runStrictChecks(root string) []string {
	var errs []string
	compiledDir := filepath.Join(root, ".compiled")

	requiredArtifacts := []string{".index.gz", ".domain.gz", ".config.gz", ".manifest.gz"}
	for _, name := range requiredArtifacts {
		artifactPath := filepath.Join(compiledDir, name)
		if _, statErr := os.Stat(artifactPath); os.IsNotExist(statErr) {
			errs = append(errs, fmt.Sprintf("strict: missing compiled artifact %s — run strategist compile", name))
		}
	}

	drift, err := compile.VerifyManifest(compiledDir)
	if err != nil {
		errs = append(errs, fmt.Sprintf("strict: verify manifest: %v", err))
	}
	for _, d := range drift {
		errs = append(errs, "strict: "+d)
	}

	return errs
}

// printSimulateReport prints the --simulate readiness report. It performs no
// provider invocation and no workspace mutation — it only reads the already
// materialized errs computed by the caller's checks and reports them as a
// per-slot/persona readiness table instead of the terse pass/fail banner.
func printSimulateReport(root string, providers map[string]string, mode, decisionReason string, errs []string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if err := writeSimulateReport(w, root, providers, mode, decisionReason, errs); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("check --simulate: flush output: %w", err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("[Strategist] check=failed errors=%d root=%s (simulate)", len(errs), root)
	}
	return nil
}

// simReportWriter accumulates tabwriter output, short-circuiting after the
// first write error so callers can emit a flat sequence of lines without a
// branch per line.
type simReportWriter struct {
	w   *tabwriter.Writer
	err error
}

func (s *simReportWriter) line(format string, args ...any) {
	if s.err != nil {
		return
	}
	if _, err := fmt.Fprintf(s.w, format, args...); err != nil {
		s.err = fmt.Errorf("check --simulate: write output: %w", err)
	}
}

// writeSimulateReport writes the --simulate readiness table to w, returning the
// first write error encountered (if any) wrapped with context.
func writeSimulateReport(w *tabwriter.Writer, root string, providers map[string]string, mode, decisionReason string, errs []string) error {
	sw := &simReportWriter{w: w}

	sw.line("READINESS\t\n")
	sw.line("  root\t%s\n", root)
	sw.line("  pipeline_route\tmain\n")
	sw.line("  decision_reason\t%s\n", decisionReason)
	sw.line("\t\n")
	sw.line("SLOTS\t\n")
	writeSimulateSlots(sw, providers)
	sw.line("\t\n")
	sw.line("PERSONA\t\n")
	sw.line("  mode\t%s\n", mode)
	writeSimulateBlockers(sw, errs)

	return sw.err
}

func writeSimulateSlots(sw *simReportWriter, providers map[string]string) {
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		status := "ready"
		if providers[slot] == "" {
			status = "missing_provider"
		}
		sw.line("  %-12s\tprovider=%s\tstatus=%s\n", slot, providers[slot], status)
	}
}

func writeSimulateBlockers(sw *simReportWriter, errs []string) {
	if len(errs) == 0 {
		return
	}
	sw.line("\t\n")
	sw.line("BLOCKERS\t\n")
	for _, e := range errs {
		sw.line("  ✗\t%s\n", e)
	}
}

func validateRuntimeDefaultParity(root string) []string {
	extractor := embedpkg.Extractor{}
	var errs []string
	manifest, manifestLoaded, manifestErr := readInstallManifest(root)
	if manifestErr != nil {
		errs = append(errs, fmt.Sprintf("runtime_stale: install manifest unreadable: %v", manifestErr))
	}

	for _, rel := range domain.NormativeRuntimeDefaultPaths() {
		err, ok := validateRuntimeDefaultFile(root, rel, extractor, manifest, manifestLoaded, manifestErr)
		if ok {
			errs = append(errs, err)
		}
	}

	return errs
}

func validateRuntimeDefaultFile(
	root, rel string,
	extractor embedpkg.Extractor,
	manifest domain.InstallManifest,
	manifestLoaded bool,
	manifestErr error,
) (string, bool) {
	runtimePath := filepath.Join(root, filepath.FromSlash(rel))
	runtimeRaw, readErr := os.ReadFile(runtimePath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return "", false
		}
		return fmt.Sprintf("runtime_stale: read %s: %v", runtimePath, readErr), true
	}

	embeddedRaw, embedErr := extractor.ReadFile(rel)
	if embedErr != nil {
		return fmt.Sprintf("runtime_stale: embedded default %q unreadable: %v", rel, embedErr), true
	}
	if string(runtimeRaw) == string(embeddedRaw) {
		return "", false
	}
	return domain.FormatRuntimeStaleDiagnostic(
		rel,
		classifyRuntimeStale(runtimeRaw, rel, manifest, manifestLoaded, manifestErr),
	), true
}

func classifyRuntimeStale(
	runtimeRaw []byte,
	rel string,
	manifest domain.InstallManifest,
	manifestLoaded bool,
	manifestErr error,
) domain.RuntimeDefaultDecision {
	if manifestErr != nil || !manifestLoaded {
		return domain.RuntimeDecisionUnknownManifest
	}
	manifestFile, ok := manifest.FileByPath(rel)
	if !ok {
		return domain.RuntimeDecisionUnknownManifest
	}
	if domain.SHA256Hex(runtimeRaw) == manifestFile.SHA256 {
		return domain.RuntimeDecisionAutoUpgrade
	}
	return domain.RuntimeDecisionConflict
}

func readInstallManifest(root string) (domain.InstallManifest, bool, error) {
	path := filepath.Join(root, domain.InstallManifestRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.InstallManifest{}, false, nil
		}
		return domain.InstallManifest{}, false, fmt.Errorf("read install manifest: %w", err)
	}
	var manifest domain.InstallManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return domain.InstallManifest{}, false, fmt.Errorf("parse install manifest: %w", err)
	}
	return manifest, true, nil
}

// printContentByLang extracts personas.<persona>.content_by_lang.<lang> and, when present,
// personas.<persona>.phase_announcements.<lang> from the compiled config artifact and prints
// both as indented JSON to stdout. This is the supported way for an agent to read localized
// runtime message templates — persona YAML source under personas/<mode>.yaml only ever
// contains the canonical English content; non-English variants (e.g. pt-BR) are injected only
// at `strategist compile` time (see internal/compile/config.go's injectPTBRRuntime) and exist
// solely in the compiled artifact. content_by_lang is required — its absence or a missing lang
// key is an error. phase_announcements is optional — some personas (e.g. pragmatic) declare no
// phase_announcements at all, so its absence is not an error; when present but the requested
// lang key is missing, it is omitted from the output rather than failing the whole call.
func printContentByLang(root, persona, lang string) error {
	if persona == "" {
		return fmt.Errorf("[Strategist] check=blocked reason=persona_not_resolved\n→ pass --persona or ensure active.yaml has a non-empty mode")
	}

	artifact, err := readCompiledContentArtifact(root)
	if err != nil {
		return err
	}
	result, err := contentByLangOutput(artifact.Personas, persona, lang)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("[Strategist] check=blocked reason=marshal_error: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

type compiledContentArtifact struct {
	Personas map[string]any `json:"personas"`
}

func readCompiledContentArtifact(root string) (compiledContentArtifact, error) {
	artifactPath := filepath.Join(root, ".compiled", ".config.gz")
	f, err := os.Open(artifactPath) //nolint:gosec // G304: path derived from strategistDir
	if err != nil {
		return compiledContentArtifact{}, compiledArtifactOpenError(artifactPath, err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // read-only file; close error is not actionable

	gz, err := gzip.NewReader(f)
	if err != nil {
		return compiledContentArtifact{}, fmt.Errorf("[Strategist] check=blocked reason=compiled_artifact_corrupt: %w", err)
	}
	defer func() { _ = gz.Close() }() //nolint:errcheck // read-only reader; close error is not actionable

	var artifact compiledContentArtifact
	if err := json.NewDecoder(gz).Decode(&artifact); err != nil {
		return compiledContentArtifact{}, fmt.Errorf("[Strategist] check=blocked reason=compiled_artifact_corrupt: %w", err)
	}
	return artifact, nil
}

func compiledArtifactOpenError(path string, err error) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("[Strategist] check=blocked reason=compiled_artifact_missing artifact=%s\n→ Run: strategist compile", path)
	}
	return fmt.Errorf("[Strategist] check=blocked reason=compiled_artifact_read_error: %w", err)
}

func contentByLangOutput(personas map[string]any, persona, lang string) (map[string]any, error) {
	personaMap, err := compiledPersonaMap(personas, persona)
	if err != nil {
		return nil, err
	}
	langContent, err := requiredLangContent(personaMap, persona, lang)
	if err != nil {
		return nil, err
	}
	return addOptionalPhaseAnnouncements(map[string]any{"content_by_lang": langContent}, personaMap, lang), nil
}

func compiledPersonaMap(personas map[string]any, persona string) (map[string]any, error) {
	personaRaw, ok := personas[persona]
	if !ok {
		return nil, fmt.Errorf("[Strategist] check=blocked reason=persona_not_found persona=%s available=%s",
			persona, strings.Join(sortedKeys(personas), ","))
	}
	personaMap, ok := personaRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("[Strategist] check=blocked reason=persona_malformed persona=%s", persona)
	}
	return personaMap, nil
}

func requiredLangContent(personaMap map[string]any, persona, lang string) (any, error) {
	cbl, ok := personaMap["content_by_lang"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("[Strategist] check=blocked reason=content_by_lang_missing persona=%s", persona)
	}
	langContent, ok := cbl[lang]
	if !ok {
		return nil, fmt.Errorf("[Strategist] check=blocked reason=lang_not_found lang=%s persona=%s available=%s",
			lang, persona, strings.Join(sortedKeys(cbl), ","))
	}
	return langContent, nil
}

func addOptionalPhaseAnnouncements(result map[string]any, personaMap map[string]any, lang string) map[string]any {
	pa, ok := personaMap["phase_announcements"].(map[string]any)
	if !ok {
		return result
	}
	if paLang, ok := pa[lang]; ok {
		result["phase_announcements"] = paLang
	}
	return result
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	checkCmd.Flags().StringVar(&checkRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	checkCmd.Flags().BoolVar(&checkStrict, "strict", false, "additionally require compiled artifacts to exist and match the recorded manifest hashes")
	checkCmd.Flags().BoolVar(&checkSimulate, "simulate", false, "print a readiness report (per-slot/persona status) instead of the pass/fail banner; never invokes providers or mutates state")
	checkCmd.Flags().StringVar(&checkPrintContentByLang, "print-content-by-lang", "", "print personas.<persona>.content_by_lang.<lang> and phase_announcements.<lang> from the compiled artifact as JSON and exit; use this instead of reading persona YAML directly to resolve non-English chat templates and mission narration lines")
	checkCmd.Flags().StringVar(&checkPrintContentByLangPersona, "persona", "", "persona/mode to read with --print-content-by-lang (default: active.yaml's mode)")
	rootCmd.AddCommand(checkCmd)
}
