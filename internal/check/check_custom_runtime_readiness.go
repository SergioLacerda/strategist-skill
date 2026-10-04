package check

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/catalog"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// customRuntimeReadiness closes the green-but-unusable case: a provider that
// declares a runtime in the installed catalog needs an executable to run, so a
// non-Ranked binding must not report ready when there is neither a recorded
// private runtime nor a host executable. It only ever blocks when the catalog
// positively says a runtime is required; an absent or unreadable catalog, or a
// provider without a runtime, keeps the previous "not evaluated" result.
func customRuntimeReadiness(root, slot, provider string) domain.ReadinessCheck {
	notEvaluated := domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "dependency_lock_not_evaluated"}
	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // fixed runtime path
	if err != nil {
		return notEvaluated
	}
	stamp, ok, err := catalog.FindRankedStamp(raw, provider)
	if errors.Is(err, domain.ErrLegacyWeaponState) {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_catalog_invalid", Detail: err.Error()}
	}
	if err != nil || !ok {
		return notEvaluated
	}
	runtime := domain.NormalizeRankedRuntime(stamp.Runtime)
	switch runtime.Kind {
	case domain.RankedRuntimeNone, domain.RankedRuntimeHost, domain.RankedRuntimeEmbedded:
		// None needs nothing; a host-channel Weapon is resolved by the host
		// skill loader (ADR-0055) and an embedded Weapon runs in-process —
		// neither depends on a local executable, so this dimension has
		// nothing to evaluate for them.
		return notEvaluated
	}
	if recordedPrivateRuntimeUsable(root, slot, provider) {
		return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "runtime_private_present"}
	}
	executable := runtimeExecutableName(runtime)
	if _, lookErr := exec.LookPath(executable); lookErr == nil {
		return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "runtime_host_executable_present", Detail: executable}
	}
	return domain.ReadinessCheck{
		Status:     domain.ReadinessBlocked,
		ReasonCode: domain.ReasonRankedRuntimeExecutableMissing,
		Detail:     domain.RankedRuntimeExecutableMissingMessage(provider, executable),
	}
}

// runtimeExecutableName reports the command a runtime kind needs on PATH (or
// recorded as a private runtime): an executable kind names it explicitly via
// Entrypoint, while an OpenSpec-root runtime's bootstrap command's first word
// names it, defaulting to "openspec" when Bootstrap is unset.
func runtimeExecutableName(runtime domain.WeaponRuntime) string {
	if runtime.Kind == domain.RankedRuntimeExecutable {
		if fields := strings.Fields(runtime.Entrypoint); len(fields) > 0 {
			return fields[0]
		}
	}
	if fields := strings.Fields(runtime.Bootstrap); len(fields) > 0 {
		return fields[0]
	}
	return "openspec"
}

// recordedPrivateRuntimeUsable reports whether ranked-runtimes.yaml records a
// current-schema runtime for slot/provider whose absolute host Node and
// contained OpenSpec launcher both still exist. A launcher recorded outside
// weapon-runtime/<provider>/openspec/ is never trusted.
func recordedPrivateRuntimeUsable(root, slot, provider string) bool {
	raw, err := os.ReadFile(filepath.Join(root, domain.RankedRuntimeStatePath)) //nolint:gosec // fixed path under the selected root
	if err != nil {
		return false
	}
	state, err := domain.ParseRankedRuntimeState(raw)
	if err != nil {
		return false
	}
	entry, ok := state.Entry(slot, provider)
	if !ok || entry.Runtime == nil || !filepath.IsAbs(entry.Runtime.Node) {
		return false
	}
	script, ok := recordedScriptPath(root, provider, entry.Runtime.Script)
	return ok && regularFile(entry.Runtime.Node) && regularFile(script)
}

func regularFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}
