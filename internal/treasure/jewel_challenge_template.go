package treasure

import "fmt"

const jewelKindTemplate = "template"

// allowedChallengeTemplateTypes mirrors the union of both Handoff Challenge
// transitions' type vocabularies (internal/handoff's
// ChallengeObjective/Boundary/Classification/Gate for archivist_to_sniper,
// plus Recall/Boundary/Classification/Verdict for ranger_to_archivist) as
// of design.md § Item 2's schema sketch. Duplicated here rather than
// imported — internal/treasure must not depend on internal/handoff, a peer
// package (see internal/domain/architecture_test.go's TestLateralIsolation).
var allowedChallengeTemplateTypes = stringSet(
	"objective", "boundary", "classification", "gate",
	"recall", "verdict",
)

var allowedJewelSeverities = stringSet("low", "medium", "high")

func stringSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}

// validateJewelChallengeTemplate checks the additive pattern/challenge_template/
// severity fields: all three are scoped to kind: template jewels only, and
// a set ChallengeTemplate must carry its own required sub-fields.
func validateJewelChallengeTemplate(j Jewel) error {
	if j.Kind != jewelKindTemplate {
		return validateChallengeTemplateFieldsAbsent(j)
	}
	if j.Severity != "" {
		if _, ok := allowedJewelSeverities[j.Severity]; !ok {
			return fmt.Errorf("jewel %q has invalid severity %q", j.ID, j.Severity)
		}
	}
	if j.ChallengeTemplate == nil {
		return nil
	}
	return validateChallengeTemplateContent(j.ID, *j.ChallengeTemplate)
}

func validateChallengeTemplateFieldsAbsent(j Jewel) error {
	if j.Pattern != "" || j.ChallengeTemplate != nil || j.Severity != "" {
		return fmt.Errorf("jewel %q sets pattern/challenge_template/severity but kind is %q, not %q", j.ID, j.Kind, jewelKindTemplate)
	}
	return nil
}

func validateChallengeTemplateContent(jewelID string, ct ChallengeTemplate) error {
	if len(ct.AppliesTo) == 0 {
		return fmt.Errorf("jewel %q challenge_template.applies_to is required", jewelID)
	}
	if ct.Type == "" {
		return fmt.Errorf("jewel %q challenge_template.type is required", jewelID)
	}
	if _, ok := allowedChallengeTemplateTypes[ct.Type]; !ok {
		return fmt.Errorf("jewel %q challenge_template.type %q is not an allowed value", jewelID, ct.Type)
	}
	if ct.Prompt == "" {
		return fmt.Errorf("jewel %q challenge_template.prompt is required", jewelID)
	}
	return nil
}
