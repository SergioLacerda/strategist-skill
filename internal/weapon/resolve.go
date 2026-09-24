// Package weapon owns Role-scoped Weapon resolution and composition
// invocation. It does not own mission state, handoffs, or Approval Gate
// transitions.
package weapon

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Registry is the catalog projection used by the mission-owned resolver.
type Registry map[string]domain.WeaponManifest

// ResolvedWeapon is one Role binding: exactly one atomic or composite Weapon.
type ResolvedWeapon struct {
	Manifest   domain.WeaponManifest
	Components []ResolvedComponent
}

// ResolvedComponent is a component in deterministic dependency order.
type ResolvedComponent struct {
	Manifest  domain.WeaponManifest
	Required  bool
	DependsOn []string
}

// Resolve validates the selected Weapon and expands its declared component
// graph. The Role and slot are authoritative; a legacy canonical_role field is
// never consulted here.
func (r Registry) Resolve(id, role, slot string) (ResolvedWeapon, error) {
	manifest, err := r.selectManifest(id, role, slot)
	if err != nil {
		return ResolvedWeapon{}, err
	}
	resolved := ResolvedWeapon{Manifest: manifest}
	if manifest.Kind == domain.WeaponKindAtomic {
		return resolved, nil
	}
	resolved.Components, err = r.resolveComponents(manifest, role, slot)
	if err != nil {
		return ResolvedWeapon{}, err
	}
	return resolved, nil
}

func (r Registry) selectManifest(id, role, slot string) (domain.WeaponManifest, error) {
	manifest, ok := r[id]
	if !ok {
		return domain.WeaponManifest{}, fmt.Errorf("weapon %q was not found in catalog", id)
	}
	if err := manifest.ValidateActive(); err != nil {
		return domain.WeaponManifest{}, fmt.Errorf("resolve Weapon %q: %w", id, err)
	}
	if !contains(manifest.Roles, role) {
		return domain.WeaponManifest{}, fmt.Errorf("weapon %q is not compatible with Role %q", id, role)
	}
	if !contains(manifest.SupportedSlots, slot) {
		return domain.WeaponManifest{}, fmt.Errorf("weapon %q is not compatible with slot %q", id, slot)
	}
	return manifest, nil
}

func (r Registry) resolveComponents(manifest domain.WeaponManifest, role, slot string) ([]ResolvedComponent, error) {
	id := manifest.ID
	if manifest.Composition == nil {
		return nil, fmt.Errorf("composite Weapon %q has no composition", id)
	}
	order, err := manifest.Composition.TopologicalOrder()
	if err != nil {
		return nil, fmt.Errorf("resolve Weapon %q composition: %w", id, err)
	}
	declared := make(map[string]domain.WeaponComponent, len(manifest.Composition.Components))
	for _, component := range manifest.Composition.Components {
		declared[component.ID] = component
	}
	components := make([]ResolvedComponent, 0, len(order))
	for _, componentID := range order {
		component, err := r.resolveComponent(id, declared[componentID], role, slot)
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}
	return components, nil
}

func (r Registry) resolveComponent(parentID string, component domain.WeaponComponent, role, slot string) (ResolvedComponent, error) {
	manifest, ok := r[component.ID]
	if !ok {
		return ResolvedComponent{}, fmt.Errorf("composite Weapon %q references missing component Weapon %q", parentID, component.ID)
	}
	if err := manifest.ValidateActive(); err != nil {
		return ResolvedComponent{}, fmt.Errorf("component Weapon %q is not invocable: %w", component.ID, err)
	}
	if !contains(manifest.Roles, role) {
		return ResolvedComponent{}, fmt.Errorf("component Weapon %q is incompatible with Role %q", component.ID, role)
	}
	if !contains(manifest.SupportedSlots, slot) {
		return ResolvedComponent{}, fmt.Errorf("component Weapon %q is incompatible with slot %q", component.ID, slot)
	}
	return ResolvedComponent{
		Manifest: manifest, Required: component.Required,
		DependsOn: append([]string(nil), component.DependsOn...),
	}, nil
}

func contains(values []string, wanted string) bool {
	wanted = strings.TrimSpace(wanted)
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}
