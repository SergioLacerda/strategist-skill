package check

import "fmt"

// transitionalViewAdvisories reports every slot resolved through a hand-made compat
// view the plugin catalog does not list (DEC-013, generation N-1). It is non-blocking:
// the Weapon still resolves today, but the fallback is removed in generation N, so
// the advisory tells the operator how to stop depending on it. Slots are reported in
// pipeline order so the output is deterministic.
func transitionalViewAdvisories(providers map[string]string, resolutions map[string]slotResolution) []string {
	var advisories []string
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		if !resolutions[slot].transitionalView {
			continue
		}
		advisories = append(advisories, fmt.Sprintf(
			"[Strategist] phase=preflight status=warn reason=compat_view_uncataloged slot=%s provider=%s (resolved through a hand-made skills/%s/skill.yaml view that plugins/catalog.yaml does not list; that fallback is removed in the next runtime layout generation — list the Weapon in plugins/catalog.yaml or add it with `strategist provider add`)",
			slot, providers[slot], providers[slot]))
	}
	return advisories
}
