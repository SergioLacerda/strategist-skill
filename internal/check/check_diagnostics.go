package check

import (
	"fmt"
	"os"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/rolevalidation"
)

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
	errs = append(errs, discoveryWeaponErrors(root, providers, resolutions)...)
	errs = append(errs, checkPluginLockParity(root, providers)...)
	errs = append(errs, validateRuntimeDefaultParity(root)...)
	if err := emitF3ConflictAttributionSignals(root, cfg.BasePath, time.Now()); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ f3_conflict_signal: %v\n", err)
	}
	if checkStrict {
		errs = append(errs, runStrictChecks(root)...)
	}
	errs = append(errs, chatLanguageErrors(language)...)
	return errs, weaponBindings
}

// discoveryWeaponErrors validates the selected discovery Weapon contract when
// discovery is filled by a skill provider.
func discoveryWeaponErrors(root string, providers map[string]string, resolutions map[string]slotResolution) []string {
	discovery, ok := resolutions[string(domain.SlotDiscovery)]
	if !ok || discovery.kind != slotResolutionSkillProvider {
		return nil
	}
	if err := validateSelectedDiscoveryWeaponContract(root, providers[string(domain.SlotDiscovery)]); err != nil {
		return []string{"selected discovery Weapon: " + err.Error()}
	}
	return nil
}

// chatLanguageErrors reports a confirmed chat language that differs from the
// configured one.
func chatLanguageErrors(language *domain.PreflightLanguage) []string {
	if checkConfirmChatLanguage == "" || language == nil || language.Chat == "" || checkConfirmChatLanguage == language.Chat {
		return nil
	}
	return []string{fmt.Sprintf("chat_language_mismatch: confirmed=%s configured=%s", checkConfirmChatLanguage, language.Chat)}
}
