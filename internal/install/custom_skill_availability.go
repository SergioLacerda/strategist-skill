package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// CustomSkillAvailability is the outcome of resolving a Wizard-typed custom
// skill id that is not already a known catalog/registry entry — tasks.md
// Task 6 (.analysis/refined/20260913-embedded-skill-directory-catalog),
// distinct from validateProvider's existing non-blocking risk-mismatch
// warning. Unavailable corresponds to ADR-0029 design.md's
// "configured_unverified" wizard state: the Wizard must not silently accept
// it as ready.
type CustomSkillAvailability struct {
	Available bool
	Reason    string
}

// resolveCustomSkillAvailability attempts local_path/already_installed
// resolution (ADR-0029 design.md § Wizard behavior) for a Wizard-typed
// custom skill id against the user's own standalone workspace —
// ~/.claude/skills/<id>/, the same location installShimStep already writes
// Strategist's own shim to (see paths.go#defaultShimPath). It never searches
// external-skills-source/ (that directory is for build-time embedding, not
// an end user's own already-installed skills) and it never fabricates
// availability — a lookup failure for any reason (no home directory, no such
// skill, malformed SKILL.md) is reported as unavailable.
func resolveCustomSkillAvailability(id string) CustomSkillAvailability {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return CustomSkillAvailability{Reason: fmt.Sprintf("cannot resolve home directory: %v", err)}
	}
	dir := filepath.Join(homeDir, claudeDirName, installedProvidersDirName, id)
	if _, err := connectors.ResolveLocalPackage(dir); err != nil {
		return CustomSkillAvailability{Reason: err.Error()}
	}
	return CustomSkillAvailability{Available: true}
}

// checkCustomSkillAvailability pauses the Wizard (returns an error rather
// than silently proceeding) for any slot whose chosen provider is neither a
// known catalog/registry entry (providerRisk) nor resolvable as an
// already-installed workspace skill. A registry-known entry is never
// affected by this check, even if its risk_score is mismatched — that case
// stays validateProvider's existing non-blocking warning; this check is
// strictly for ids the registry has no opinion on at all.
func checkCustomSkillAvailability(providerRisk map[string]string, slots map[string]string) error {
	for _, slot := range sortedSlotNames(slots) {
		value := slots[slot]
		if value == "" {
			continue
		}
		if _, known := providerRisk[value]; known {
			continue
		}
		availability := resolveCustomSkillAvailability(value)
		if !availability.Available {
			return fmt.Errorf("skill %q for slot %s is unavailable (configured_unverified): %s — install it under ~/.claude/skills/%s/ first, or choose a listed provider", value, slot, availability.Reason, value)
		}
	}
	return nil
}
