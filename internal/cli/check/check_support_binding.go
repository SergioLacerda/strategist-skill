package check

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/SergioLacerda/strategist-skill/internal/runtime/catalog"
	"gopkg.in/yaml.v3"
)

// runbookFeat is the Feat whose must_invoke declaration makes a role depend
// on the JEWELER support binding.
const runbookFeat = "select_runbook"

// supportBindingReadiness reads the host environment; tests substitute it to
// keep fixture workspaces independent of the operator's JEWELER binding.
var supportBindingReadiness = func(root string) []string {
	return supportBindingAdvisories(root, os.Getenv)
}

// supportBindingAdvisories reports the readiness of the JEWELER support
// binding for every role that declares select_runbook with must_invoke: true.
// Every Role is expected in every workspace, but providers are invoked only
// inside a mission: this check inspects the binding (executable configured,
// present, and accepted against the admitted catalog Weapon) and never runs
// the provider. It is advisory and never changes readiness or exit status.
func supportBindingAdvisories(root string, getenv func(string) string) []string {
	roles := rolesRequiringRunbookFeat(root)
	if len(roles) == 0 {
		return nil
	}
	detail := jewelerBindingProblem(root, getenv(jeweler.ExecutableEnv))
	if detail == "" {
		return nil
	}
	advisories := make([]string, 0, len(roles))
	for _, role := range roles {
		advisories = append(advisories, fmt.Sprintf(
			"[Strategist] phase=preflight status=warn reason=support_binding_unready role=%s feat=%s binding=jeweler detail=%s (set %s to the admitted treasure-chest-tmp executable; the Feat fails inside a mission without it)",
			role, runbookFeat, detail, jeweler.ExecutableEnv))
	}
	return advisories
}

func jewelerBindingProblem(root, executable string) string {
	if executable == "" {
		return "executable_not_configured"
	}
	if info, err := os.Stat(executable); err != nil || info.IsDir() {
		return "executable_not_found"
	}
	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed path under the runtime root
	if err != nil {
		return "catalog_unreadable"
	}
	registry, err := catalog.ParseCompiledRegistryCatalog(raw)
	if err != nil {
		return "catalog_invalid"
	}
	identity := jeweler.AdmittedTreasureChestTMPIdentity()
	weapon, ok := registry.Weapon(identity.Package, identity.Version)
	if !ok {
		return "weapon_not_admitted"
	}
	if _, err := jeweler.NewCLIProviderFromCompiledWeapon(weapon, executable); err != nil {
		return "binding_invalid"
	}
	return ""
}

// rolesRequiringRunbookFeat lists, sorted, the roles under roles/ whose
// canonical behaviors declare select_runbook with must_invoke: true.
func rolesRequiringRunbookFeat(root string) []string {
	paths, err := filepath.Glob(filepath.Join(root, "roles", "*.yaml"))
	if err != nil {
		return nil
	}
	var roles []string
	for _, path := range paths {
		raw, err := os.ReadFile(path) //nolint:gosec // G304: role files under the runtime root
		if err != nil {
			continue
		}
		var role struct {
			Role      string      `yaml:"role"`
			Canonical []yaml.Node `yaml:"canonical"`
		}
		if yaml.Unmarshal(raw, &role) != nil || role.Role == "" {
			continue
		}
		if declaresMandatoryRunbookFeat(role.Canonical) {
			roles = append(roles, role.Role)
		}
	}
	sort.Strings(roles)
	return roles
}

func declaresMandatoryRunbookFeat(canonical []yaml.Node) bool {
	for i := range canonical {
		var entry map[string]struct {
			MustInvoke bool `yaml:"must_invoke"`
		}
		if canonical[i].Kind != yaml.MappingNode || canonical[i].Decode(&entry) != nil {
			continue
		}
		if feat, ok := entry[runbookFeat]; ok && feat.MustInvoke {
			return true
		}
	}
	return false
}
