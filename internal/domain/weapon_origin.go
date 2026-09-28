package domain

import (
	"errors"
	"fmt"
)

// ErrLegacyWeaponState marks persisted or generated state that still uses the
// pre-Weapon vocabulary. Callers fail closed on it; nothing translates it.
var ErrLegacyWeaponState = errors.New("legacy Weapon vocabulary")

// WeaponOrigin records where a Weapon package comes from, independently of
// the runtime that invokes it.
type WeaponOrigin string

// Weapon origins.
const (
	// WeaponOriginEmbedded marks a Weapon shipped in the canonical embedded source.
	WeaponOriginEmbedded WeaponOrigin = "embedded"
	// WeaponOriginCustom marks an operator-provided Weapon outside the embedded catalog.
	WeaponOriginCustom WeaponOrigin = "custom"
)

// Validate reports whether the origin is one of the canonical values.
func (o WeaponOrigin) Validate() error {
	switch o {
	case WeaponOriginEmbedded, WeaponOriginCustom:
		return nil
	default:
		return fmt.Errorf("origin %q must be embedded or custom", string(o))
	}
}

func (w WeaponManifest) validateOrigin() []string {
	if err := w.Origin.Validate(); err != nil {
		return []string{err.Error()}
	}
	return nil
}

// Legacy runtime kinds from before the strict Weapon vocabulary. They are
// recognized only to produce an actionable rejection; they are never accepted,
// translated, or written.
const (
	legacyRuntimeEmbeddedSkill = "embedded_skill"
	legacyRuntimeHostSkill     = "host_skill"
)

// unsupportedRuntimeKindError names the migration path for a legacy runtime
// kind and otherwise reports the kind as unsupported.
func unsupportedRuntimeKindError(kind string) error {
	if isLegacyRuntimeKind(kind) {
		return fmt.Errorf("%w: runtime kind %q is no longer supported; regenerate or reinstall the workspace to use the strict Weapon runtime kinds (embedded, host, executable, openspec_root)", ErrLegacyWeaponState, kind)
	}
	return fmt.Errorf("unsupported ranked runtime kind %q", kind)
}

func isLegacyRuntimeKind(kind string) bool {
	return kind == legacyRuntimeEmbeddedSkill || kind == legacyRuntimeHostSkill
}

// ValidateActiveRuntimeKind checks that kind names an invocable Weapon runtime
// kind, rejecting legacy kinds with the same migration diagnostic as a full
// runtime contract. It is for records that carry only the kind, such as locks.
func ValidateActiveRuntimeKind(kind string) error {
	switch kind {
	case RankedRuntimeEmbedded, RankedRuntimeHost, RankedRuntimeExecutable, RankedRuntimeOpenSpecRoot:
		return nil
	case RankedRuntimeNone, "":
		return fmt.Errorf("active Weapon requires an invocable runtime")
	default:
		return unsupportedRuntimeKindError(kind)
	}
}
