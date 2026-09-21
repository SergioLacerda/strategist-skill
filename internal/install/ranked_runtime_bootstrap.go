package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
	"gopkg.in/yaml.v3"
)

const rankedRuntimeStatePath = "ranked-runtimes.yaml"

type rankedRuntimeState struct {
	SchemaVersion string                    `yaml:"schema_version" json:"schema_version"`
	Entries       []rankedRuntimeStateEntry `yaml:"entries" json:"entries"`
}

// rankedRuntimeStateRuntime records the verified private runtime a Ranked
// provider executes from. Paths are slash-separated and relative to the
// Strategist root; it is absent when the provider uses a host executable.
type rankedRuntimeStateRuntime struct {
	Node       string                        `yaml:"node" json:"node"`
	Script     string                        `yaml:"script" json:"script"`
	Components []rankedRuntimeStateComponent `yaml:"components" json:"components"`
}

type rankedRuntimeStateComponent struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
	SHA256  string `yaml:"sha256" json:"sha256"`
}

type rankedRuntimeStateEntry struct {
	Role           string `yaml:"role" json:"role"`
	Slot           string `yaml:"slot" json:"slot"`
	Provider       string `yaml:"provider" json:"provider"`
	ContractDigest string `yaml:"contract_digest" json:"contract_digest"`
	Root           string `yaml:"root,omitempty" json:"root,omitempty"`
	Kind           string `yaml:"kind" json:"kind"`

	Runtime *rankedRuntimeStateRuntime `yaml:"runtime,omitempty" json:"runtime,omitempty"`
}

// prepareRankedProviderRuntimes materializes the runtime contract of each
// selected Ranked slot after active.yaml/plugins.lock have been persisted.
// It intentionally reads the installed catalog, not the build checkout, so
// the installed binding and runtime cannot drift silently.
func prepareRankedProviderRuntimes(ctx context.Context, strategistDir string) error {
	rankedBindings, err := loadRankedBindings(strategistDir)
	if err != nil {
		return err
	}
	if len(rankedBindings) == 0 {
		return nil
	}
	roles, catalog, err := loadRankedRuntimeInputs(strategistDir)
	if err != nil {
		return err
	}
	state, err := prepareRankedRuntimeState(ctx, strategistDir, roles, catalog, rankedBindings)
	if err != nil {
		return err
	}
	return writeRankedRuntimeState(strategistDir, state)
}

func prepareRankedRuntimeState(ctx context.Context, strategistDir string, roles domain.RoleSlotMap, catalog pluginCatalog, bindings []domain.SlotBinding) (rankedRuntimeState, error) {
	state := rankedRuntimeState{SchemaVersion: "strategist-ranked-runtime/v1"}
	for _, binding := range bindings {
		entry, required, err := prepareRankedBinding(ctx, strategistDir, roles, catalog, binding)
		if err != nil {
			return rankedRuntimeState{}, err
		}
		if required {
			state.Entries = append(state.Entries, entry)
		}
	}
	return state, nil
}

func loadRankedBindings(strategistDir string) ([]domain.SlotBinding, error) {
	lockRaw, err := os.ReadFile(filepath.Join(strategistDir, "plugins.lock")) //nolint:gosec // fixed path under the install root
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugins.lock for ranked runtime: %w", err)
	}
	var lock domain.PluginLockFile
	if err := yaml.Unmarshal(lockRaw, &lock); err != nil {
		return nil, fmt.Errorf("parse plugins.lock for ranked runtime: %w", err)
	}
	bindings := make([]domain.SlotBinding, 0, len(lock.Bindings))
	for _, binding := range lock.Bindings {
		if binding.EffectiveMode() == domain.SlotBindingModeRanked {
			bindings = append(bindings, binding)
		}
	}
	return bindings, nil
}

func loadRankedRuntimeInputs(strategistDir string) (domain.RoleSlotMap, pluginCatalog, error) {
	roleRaw, err := os.ReadFile(filepath.Join(strategistDir, "roles", "default.yaml")) //nolint:gosec // fixed path under the install root
	if err != nil {
		return nil, pluginCatalog{}, fmt.Errorf("read role map for ranked runtime: %w", err)
	}
	var roles domain.RoleSlotMap
	if err := yaml.Unmarshal(roleRaw, &roles); err != nil {
		return nil, pluginCatalog{}, fmt.Errorf("parse role map for ranked runtime: %w", err)
	}
	catalogRaw, err := os.ReadFile(filepath.Join(strategistDir, pluginCatalogPath)) //nolint:gosec // fixed path under the install root
	if err != nil {
		return nil, pluginCatalog{}, fmt.Errorf("read catalog for ranked runtime: %w", err)
	}
	catalog, err := parseCatalogBytes(catalogRaw)
	if err != nil {
		return nil, pluginCatalog{}, fmt.Errorf("parse catalog for ranked runtime: %w", err)
	}
	return roles, catalog, nil
}

func prepareRankedBinding(ctx context.Context, strategistDir string, roles domain.RoleSlotMap, catalog pluginCatalog, binding domain.SlotBinding) (rankedRuntimeStateEntry, bool, error) {
	provider, runtime, err := resolveRankedProvider(catalog, binding)
	if err != nil || runtime.Kind == domain.RankedRuntimeNone {
		return rankedRuntimeStateEntry{}, false, err
	}
	private, err := bootstrapRankedProvider(ctx, strategistDir, provider, runtime)
	if err != nil {
		return rankedRuntimeStateEntry{}, false, err
	}
	return rankedRuntimeStateEntry{
		Runtime:        private,
		Role:           roles[binding.Slot],
		Slot:           binding.Slot,
		Provider:       provider.ID,
		ContractDigest: provider.CertificationDigest,
		Root:           runtime.Root,
		Kind:           runtime.Kind,
	}, true, nil
}

func bootstrapOpenSpecRuntime(ctx context.Context, root string, runtime domain.RankedRuntimeContract) error {
	return bootstrapOpenSpecRuntimeWith(ctx, root, runtime, hostOpenSpec)
}

func bootstrapOpenSpecRuntimeWith(ctx context.Context, root string, runtime domain.RankedRuntimeContract, exe rankedExecutable) error {
	bootstrapArgs, err := openSpecCommandArgs(runtime.Bootstrap, "init")
	if err != nil {
		return fmt.Errorf("invalid bootstrap command: %w", err)
	}
	healthcheckArgs, err := openSpecCommandArgs(runtime.Healthcheck, "context")
	if err != nil {
		return fmt.Errorf("invalid healthcheck command: %w", err)
	}
	if runtimefs.Exists(filepath.Join(root, "config.yaml")) {
		return validateExistingOpenSpecRuntime(ctx, root, exe, healthcheckArgs)
	}
	return initializeOpenSpecRuntime(ctx, root, exe, bootstrapArgs, healthcheckArgs)
}

func validateExistingOpenSpecRuntime(ctx context.Context, root string, exe rankedExecutable, healthcheckArgs []string) error {
	output, err := runRankedRuntimeCommand(ctx, root, exe.name, exe.args(healthcheckArgs)...)
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}
	if err := domain.ValidateOpenSpecHealthcheck(output, root); err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}
	return removeLegacyNestedOpenSpecRoot(root)
}

func initializeOpenSpecRuntime(ctx context.Context, root string, exe rankedExecutable, bootstrapArgs, healthcheckArgs []string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create root %s: %w", root, err)
	}
	if _, err := runRankedRuntimeCommand(ctx, filepath.Dir(root), exe.name, exe.args(bootstrapArgs)...); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	if !runtimefs.Exists(filepath.Join(root, "config.yaml")) {
		return fmt.Errorf("bootstrap completed without %s/config.yaml", root)
	}
	return validateExistingOpenSpecRuntime(ctx, root, exe, healthcheckArgs)
}
