package dojo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type skeleton struct {
	Scenario    string                 `yaml:"scenario"`
	Description string                 `yaml:"description"`
	Kind        string                 `yaml:"kind"`
	Calibration domain.DojoCalibration `yaml:"calibration"`
}

func newGenerate(deps Dependencies) *cobra.Command {
	var root, boundary, layer, profile, gateRule string
	var include []string
	var synthetic bool
	var packages, repetitions int
	var exclude, localScope []string
	cmd := &cobra.Command{
		Use:   "generate <case>",
		Short: "Write the skeleton of a judge calibration case",
		Long: `Write <base_path>/dojo/<case>/criteria.yaml for a judge_calibration case with its defaults and an
explicit risk policy. The skeleton holds no package content and never overwrites an existing case.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(cmd, deps, generateRequest{root: root, name: args[0], boundary: boundary, layer: layer, profile: profile, gateRule: gateRule, include: include, synthetic: synthetic, packages: packages, repetitions: repetitions, exclude: exclude, localScope: localScope})
		},
	}
	cmd.Flags().StringVar(&root, "root", "", rootFlagUsage)
	cmd.Flags().StringVar(&boundary, "boundary", calibration.BoundaryPackage, "boundary to evaluate: package or ranger")
	cmd.Flags().StringVar(&layer, "layer", "judge", "judge (the real provider, needs consent), integration (the approval policy against scripted answers, no call) or live (the provider smoke check, needs consent)")
	cmd.Flags().StringVar(&profile, "profile", "", "question profile: handoff-v1 (default), handoff-v2 (a missing field is shown as (absent)) or handoff-atomic-v1 (one question per criterion, the lowest answer decides)")
	cmd.Flags().StringVar(&gateRule, "gate-rule", "", "gate rule of the case: gates-2 (default) or gates-3 (cross-validated, pre-registered for the real corpus)")
	cmd.Flags().BoolVar(&synthetic, "synthetic", false, "write synthetic fixture packages under the case and calibrate over them: golden by construction, nothing real is sent")
	cmd.Flags().IntVar(&packages, "packages", 8, "with --synthetic: how many fixture packages to write (up to 32)")
	cmd.Flags().IntVar(&repetitions, "repetitions", 0, "repetitions per case (0 keeps the default of 3)")
	cmd.Flags().StringSliceVar(&localScope, "local-scope", nil, "delegate this operator's protection to the local checks: the judge gates skip it and a local coverage gate must pass for it")
	cmd.Flags().StringSliceVar(&exclude, "exclude-operator", nil, "leave an operator out of the case (recorded in the case and in its calibration)")
	cmd.Flags().StringSliceVar(&include, "package", nil, "restrict the corpus to this package (repeatable)")
	return cmd
}

type generateRequest struct {
	root, name, boundary, layer, profile, gateRule string
	include                                        []string
	synthetic                                      bool
	packages                                       int
	repetitions                                    int
	exclude                                        []string
	localScope                                     []string
}

func runGenerate(cmd *cobra.Command, deps Dependencies, req generateRequest) error {
	_, basePath, err := resolveRoots(deps, req.root)
	if err != nil {
		return err
	}
	paths, err := internaldojo.NewStoragePaths(basePath, req.name)
	if err != nil {
		return fmt.Errorf("dojo generate: %w", err)
	}
	file := filepath.Join(paths.ScenarioDir, "criteria.yaml")
	if _, err := os.Stat(file); err == nil {
		return fmt.Errorf("dojo generate: %s already exists; a case is never overwritten", file)
	}
	doc := buildSkeleton(req)
	if err := internaldojo.ValidateCriteria(domain.DojoCriteria{Scenario: doc.Scenario, Kind: doc.Kind, Calibration: &doc.Calibration}); err != nil {
		return fmt.Errorf("dojo generate: %w", err)
	}
	if err := writeSkeleton(paths.ScenarioDir, file, doc, req, basePath); err != nil {
		return err
	}
	return say(cmd.OutOrStdout(), "wrote %s\nnext: strategist dojo calibrate %s --dry-run\n", file, req.name)
}

// buildSkeleton is the case document a request describes, before it is validated.
func buildSkeleton(req generateRequest) skeleton {
	doc := newSkeleton(req.name, req.boundary, req.layer, req.include)
	doc.Calibration.Profile = req.profile
	doc.Calibration.GateRule = req.gateRule
	if req.repetitions > 0 {
		doc.Calibration.Repetitions = req.repetitions
	}
	doc.Calibration.Operators = operatorsWithout(req.boundary, req.exclude)
	doc.Calibration.LocalScope = req.localScope
	if req.layer == "pair" {
		doc.Calibration.Boundary = calibration.BoundaryPairName
		doc.Description = "Pair laboratory: atomic requirement-to-task questions, one dimension at a time, over synthetic pairs whose oracle is known by construction; nothing real is sent."
	}
	if req.synthetic {
		doc.Calibration.Corpus.Roots = []string{filepath.ToSlash(filepath.Join("dojo", req.name, "fixtures", "done"))}
		doc.Calibration.Corpus.Synthetic = true
		doc.Description = "Synthetic pilot: calibrates the pre-check over synthetic fixture packages that are golden by construction; nothing real is sent."
	}
	return doc
}

func writeSkeleton(dir, file string, doc skeleton, req generateRequest, basePath string) error {
	raw, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("dojo generate: encode: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: dojo storage domain
		return fmt.Errorf("dojo generate: create %s: %w", dir, err)
	}
	if err := os.WriteFile(file, raw, 0o644); err != nil { //nolint:gosec // G306: dojo storage domain
		return fmt.Errorf("dojo generate: write %s: %w", file, err)
	}
	return writeFixtures(dir, doc, req, basePath)
}

// writeFixtures writes the fixtures a case kind needs next to its criteria.
func writeFixtures(dir string, doc skeleton, req generateRequest, basePath string) error {
	var err error
	switch {
	case req.layer == "pair":
		err = internaldojo.WritePairs(filepath.Join(dir, "pairs.yaml"), calibration.SyntheticPairs(req.packages))
	case req.synthetic:
		_, err = internaldojo.WriteSyntheticPackages(filepath.Join(basePath, filepath.FromSlash(doc.Calibration.Corpus.Roots[0])), req.packages)
	}
	if err != nil {
		return fmt.Errorf("dojo generate: %w", err)
	}
	return nil
}

func newSkeleton(name, boundary, layer string, include []string) skeleton {
	return skeleton{
		Scenario: name, Kind: domain.KindJudgeCalibration,
		Description: describe(boundary, layer),
		Calibration: domain.DojoCalibration{
			Layer: layer, Boundary: boundary, Provider: "jev", Stage: 1, Seed: 1, Repetitions: internaldojo.DefaultRepetitions,
			RiskPolicy: domain.DojoRiskPolicy{MaxFalseApproval: calibration.DefaultPolicy().MaxFalseApproval, MaxFalseRejection: calibration.DefaultPolicy().MaxFalseRejection},
			Corpus:     domain.DojoCorpus{Include: include, HeldOutPercent: internaldojo.DefaultHeldOutPercent},
			MaxAgeDays: internaldojo.DefaultMaxAgeDays,
		},
	}
}

func describe(boundary, layer string) string {
	if layer == "live" {
		return "Live smoke check: authenticates and asks the pre-check two synthetic handoffs; synthetic text only, behind a consent term."
	}
	if layer == "integration" {
		return "Integration layer: the approval policy (threshold, fallback, delegation, pinned calibration) against scripted answers; no provider is called."
	}
	return "Judge calibration case: measures the provider handoff pre-check on the " + boundary + " boundary against labeled cases."
}

// operatorsWithout lists the stage 1 operators of a boundary minus the excluded
// ones, or nothing (the whole catalog) when none is excluded. An unknown name is
// ignored here and refused by the case validation of the next step.
func operatorsWithout(boundary string, exclude []string) []string {
	if len(exclude) == 0 {
		return nil
	}
	skip := map[string]bool{}
	for _, name := range exclude {
		skip[name] = true
	}
	var names []string
	for _, op := range calibration.Catalog(boundary) {
		if op.Stage <= 1 && !skip[op.Name] {
			names = append(names, op.Name)
		}
	}
	return names
}
