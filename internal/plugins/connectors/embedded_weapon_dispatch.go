package connectors

import (
	"context"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// EmbeddedRuntimeDependencies are the only capabilities production dispatch
// may use for Ranked Embedded Weapons.
type EmbeddedRuntimeDependencies struct {
	CodeInvokers map[string]EmbeddedWeaponInvoker
	PromptBridge EmbeddedPromptBridge
	Payloads     EmbeddedPromptPayloadSource
}

// NewProductionEmbeddedDispatch creates dispatch entries from the compiled
// registry. It never discovers an invoker by path or provider ID alone.
func NewProductionEmbeddedDispatch(registry domain.CompiledRegistry, deps EmbeddedRuntimeDependencies) (EmbeddedWeaponDispatch, error) {
	invokers := make(map[string]EmbeddedWeaponInvoker)
	for _, weapon := range registry.Weapons {
		if weapon.Runtime.Kind != domain.RankedRuntimeEmbedded {
			continue
		}
		invoker, err := productionEmbeddedInvoker(registry, weapon, deps)
		if err != nil {
			return EmbeddedWeaponDispatch{}, err
		}
		invokers[weapon.Identity()] = invoker
	}
	return NewEmbeddedWeaponDispatch(invokers)
}

func productionEmbeddedInvoker(registry domain.CompiledRegistry, weapon domain.CompiledWeapon, deps EmbeddedRuntimeDependencies) (EmbeddedWeaponInvoker, error) {
	switch weapon.Runtime.ExecutionMode {
	case domain.WeaponExecutionModeCode:
		return productionCodeInvoker(weapon.ID, deps.CodeInvokers)
	case domain.WeaponExecutionModePromptBridge:
		return productionPromptInvoker(registry, weapon, deps)
	default:
		return nil, fmt.Errorf("embedded_weapon_dispatch_unavailable: Weapon %q has unsupported execution mode %q", weapon.ID, weapon.Runtime.ExecutionMode)
	}
}

func productionCodeInvoker(weaponID string, codeInvokers map[string]EmbeddedWeaponInvoker) (EmbeddedWeaponInvoker, error) {
	invoker := codeInvokers[weaponID]
	if invoker == nil {
		return nil, fmt.Errorf("embedded_weapon_dispatch_unavailable: code invoker for %q is not registered", weaponID)
	}
	return invoker, nil
}

func productionPromptInvoker(registry domain.CompiledRegistry, weapon domain.CompiledWeapon, deps EmbeddedRuntimeDependencies) (EmbeddedWeaponInvoker, error) {
	if deps.PromptBridge == nil || deps.Payloads == nil {
		return nil, fmt.Errorf("embedded_weapon_dispatch_unavailable: prompt bridge for %q is not registered", weapon.ID)
	}
	bindingDigest := rankedBindingDigest(registry, weapon)
	if bindingDigest == "" {
		return nil, fmt.Errorf("embedded_weapon_dispatch_unavailable: Ranked binding for %q is not registered", weapon.Identity())
	}
	return NewEmbeddedPromptInvoker(weapon.ID, weapon.Version, weapon.Entrypoint, bindingDigest, weapon.SourceDigest, deps.Payloads, deps.PromptBridge), nil
}

func rankedBindingDigest(registry domain.CompiledRegistry, weapon domain.CompiledWeapon) string {
	for _, binding := range registry.RankedBindings {
		if binding.WeaponID == weapon.ID && binding.WeaponVersion == weapon.Version {
			return binding.BindingDigest
		}
	}
	return ""
}

// EmbeddedWeaponDispatch is the explicit Strategist-owned dispatch table for
// Ranked Embedded Weapons. It is intentionally constructed by the runtime
// adapter with real invokers; catalog metadata alone never creates authority.
type EmbeddedWeaponDispatch struct {
	ConnectorAPIVersion string
	Invokers            map[string]EmbeddedWeaponInvoker
}

// NewEmbeddedWeaponDispatch validates and copies the invoker table so callers
// cannot mutate runtime authority after construction.
func NewEmbeddedWeaponDispatch(invokers map[string]EmbeddedWeaponInvoker) (EmbeddedWeaponDispatch, error) {
	if len(invokers) == 0 {
		return EmbeddedWeaponDispatch{}, fmt.Errorf("embedded weapon dispatch: at least one invoker is required")
	}
	copyOfInvokers := make(map[string]EmbeddedWeaponInvoker, len(invokers))
	for weaponID, invoker := range invokers {
		if strings.TrimSpace(weaponID) == "" {
			return EmbeddedWeaponDispatch{}, fmt.Errorf("embedded weapon dispatch: Weapon ID is required")
		}
		if invoker == nil {
			return EmbeddedWeaponDispatch{}, fmt.Errorf("embedded weapon dispatch: invoker for %q is nil", weaponID)
		}
		copyOfInvokers[weaponID] = invoker
	}
	return EmbeddedWeaponDispatch{ConnectorAPIVersion: "strategist-connector-api/1", Invokers: copyOfInvokers}, nil
}

// Connector resolves one compiled Ranked Weapon version to an Embedded
// connector. The table is keyed by "id@version"; a bare id or an unknown
// version never resolves. Missing dispatch is an explicit error; it never
// falls back to a host or filesystem loader.
func (d EmbeddedWeaponDispatch) Connector(weaponID, version string) (RuntimeConnector, error) {
	identity := domain.WeaponIdentity(weaponID, version)
	invoker, ok := d.Invokers[identity]
	if version == "" || !ok || invoker == nil {
		return nil, fmt.Errorf("embedded_weapon_dispatch_unavailable: Weapon %q has no registered invoker", identity)
	}
	return EmbeddedWeaponConnector{
		ConnectorID:         "strategist-embedded/" + identity,
		ConnectorAPIVersion: d.ConnectorAPIVersion,
		Invoker:             invoker,
	}, nil
}

// Probe returns the readiness result for a compiled internal Weapon without
// pretending that a missing invoker is statically executable.
func (d EmbeddedWeaponDispatch) Probe(ctx context.Context, weaponID, version, entrypoint string) ConnectorResult {
	connector, err := d.Connector(weaponID, version)
	if err != nil {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "embedded_weapon_dispatch_unavailable", Detail: err.Error()}
	}
	return connector.Probe(ctx, domain.InstalledInstance{ID: weaponID, ConnectorID: "strategist-embedded/" + domain.WeaponIdentity(weaponID, version)}, entrypoint)
}
