package leveling

import (
	"fmt"
	"strings"
)

// ValidEffortTier reports whether effort is one of the policy's effort tiers.
// The catalog is shared with domain.LevelingEffortTiers (a parity test guards
// drift), so the manual and host paths accept exactly the same values.
func ValidEffortTier(effort string) bool {
	return effortTiers[strings.ToLower(strings.TrimSpace(effort))]
}

// isPlaceholder reports whether a value is still the literal reminder carried
// by the role `on_start` template (`<your-model>`, `<your-effort>`). Running
// the hook verbatim must degrade to an unlabelled line, not record the
// placeholder as if it were the running model.
func isPlaceholder(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "<") && strings.HasSuffix(value, ">") && len(value) > 1
}

// SanitizeHost drops host-reported values that cannot be trusted and explains
// each rejection. Dropping a field is never fatal: resolution falls back to the
// remaining sources and, failing those, to the unlabelled form.
func SanitizeHost(host Host) (Host, []string) {
	model, modelRejection := sanitizeHostModel(host.Model)
	effort, effortRejection := sanitizeHostEffort(host.Effort)
	rejected := make([]string, 0, 2)
	for _, rejection := range []string{modelRejection, effortRejection} {
		if rejection != "" {
			rejected = append(rejected, rejection)
		}
	}
	return Host{Model: model, Effort: effort}, rejected
}

func sanitizeHostModel(model string) (string, string) {
	model = strings.TrimSpace(model)
	switch {
	case model == "":
		return "", ""
	case isPlaceholder(model):
		return "", fmt.Sprintf("ignored --host-model %q: the on_start placeholder was not replaced with the running model", model)
	case strings.ContainsAny(model, "\r\n"):
		return "", fmt.Sprintf("ignored --host-model %q: a model name must be a single line", model)
	}
	return model, ""
}

func sanitizeHostEffort(effort string) (string, string) {
	effort = strings.ToLower(strings.TrimSpace(effort))
	switch {
	case effort == "":
		return "", ""
	case isPlaceholder(effort):
		return "", fmt.Sprintf("ignored --host-effort %q: the on_start placeholder was not replaced with the running effort", effort)
	case !ValidEffortTier(effort):
		return "", fmt.Sprintf("ignored --host-effort %q: not one of %s", effort, strings.Join(EffortTierNames(), ", "))
	}
	return effort, ""
}

// EffortTierNames lists the accepted effort tiers in ascending order, for use
// in diagnostics and command help.
func EffortTierNames() []string {
	return append([]string(nil), orderedEffortTiers...)
}
