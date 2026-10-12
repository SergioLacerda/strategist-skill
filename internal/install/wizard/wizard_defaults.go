package wizard

import (
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/support"
	"gopkg.in/yaml.v3"
)

// Defaults are the values of an existing configuration, used as the
// wizard's defaults so that accepting every default keeps the current
// workspace instead of resetting it to the catalog defaults (a re-run used to
// reset the slots). The zero value means "no existing configuration" and
// leaves every prompt on its catalog default.
type Defaults struct {
	UI, Docs, Chat, Code string
	Mode, BasePath       string
	ADRRoot, RunbookRoot string
	Slots                map[string]string // slot -> provider as written in active.yaml
	Modes                map[string]string // slot -> ranked | custom, from plugins.lock
}

// LoadWizardDefaults reads active.yaml and the binding modes of plugins.lock.
// Any read problem yields the zero value: the wizard must still run.
func LoadWizardDefaults(strategistDir string) Defaults {
	raw, err := os.ReadFile(filepath.Join(strategistDir, names.ActiveYAMLName)) //nolint:gosec // fixed file under the runtime dir
	if err != nil {
		return Defaults{}
	}
	var cfg struct {
		Mode     string            `yaml:"mode"`
		BasePath string            `yaml:"base_path"`
		Language map[string]string `yaml:"language"`
		Slots    map[string]string `yaml:"slots"`
		ADR      struct {
			CanonicalPath string `yaml:"canonical_path"`
		} `yaml:"adr"`
		Runbooks struct {
			CanonicalPath string `yaml:"canonical_path"`
		} `yaml:"runbooks"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Defaults{}
	}
	d := Defaults{
		UI: cfg.Language["ui"], Docs: cfg.Language["docs"], Chat: cfg.Language["chat"], Code: cfg.Language["code"],
		Mode: cfg.Mode, BasePath: cfg.BasePath,
		ADRRoot: cfg.ADR.CanonicalPath, RunbookRoot: cfg.Runbooks.CanonicalPath,
		Slots: cfg.Slots, Modes: map[string]string{},
	}
	if lock, err := plugincatalog.ReadPluginLockFile(strategistDir); err == nil {
		for _, binding := range lock.Bindings {
			d.Modes[binding.Slot] = binding.EffectiveMode()
		}
	}
	return d
}

// OrDefault returns value when it is set and valid for options (or options is
// empty), else the catalog default.
func OrDefault(value, catalogDefault string, options []string) string {
	if value == "" {
		return catalogDefault
	}
	if len(options) > 0 && !support.ContainsString(options, value) {
		return catalogDefault
	}
	return value
}

// applyToSlot moves the pre-selection of a slot's options to the existing
// binding when that Weapon is still offered; otherwise the options are
// unchanged. A Custom binding is preselected as Custom, never as Ranked.
func (d Defaults) applyToSlot(options plugincatalog.SlotOptions, slot string) plugincatalog.SlotOptions {
	existing := d.Slots[slot]
	if existing == "" {
		return options
	}
	if d.Modes[slot] != domain.SlotBindingModeCustom {
		if ref, ok := MatchingOption(options.RankedRefs, existing); ok {
			options.RankedDefault = ref
			return options
		}
	}
	if candidate, ok := MatchingOption(options.IDs, existing); ok {
		options.DefaultID = candidate
		options.PreferCustom = true
	}
	return options
}

// MatchingOption finds the offered reference that names the existing Weapon,
// by exact reference or by id.
func MatchingOption(offered []string, existing string) (string, bool) {
	id := DomainProviderID(existing)
	for _, ref := range offered {
		if ref == existing || DomainProviderID(ref) == id {
			return ref, true
		}
	}
	return "", false
}
