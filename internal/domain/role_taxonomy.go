package domain

import (
	"fmt"
	"strings"
)

// RoleOrigin identifies who owns the role contract. It is independent from
// whether a provider may fill the role.
type RoleOrigin string

const (
	// RoleOriginNative marks a role whose contract is owned by strategist.
	RoleOriginNative RoleOrigin = "native"
	// RoleOriginExternal marks a role whose contract is owned outside strategist.
	RoleOriginExternal RoleOrigin = "external"
)

// Validate rejects role origins that are not part of the public taxonomy.
func (o RoleOrigin) Validate() error {
	if o != RoleOriginNative && o != RoleOriginExternal {
		return fmt.Errorf("role origin %q is not one of native, external", o)
	}
	return nil
}

// RoleExtensibility identifies whether a role contract accepts a provider.
type RoleExtensibility string

const (
	// RoleExtensibilityFixed marks a role that never accepts a provider.
	RoleExtensibilityFixed RoleExtensibility = "fixed"
	// RoleExtensibilityPluggable marks a role that accepts a provider.
	RoleExtensibilityPluggable RoleExtensibility = "pluggable"
)

// Validate rejects role extensibility values that are not part of the public
// taxonomy.
func (e RoleExtensibility) Validate() error {
	if e != RoleExtensibilityFixed && e != RoleExtensibilityPluggable {
		return fmt.Errorf("role extensibility %q is not one of fixed, pluggable", e)
	}
	return nil
}

// IsPluggable reports whether a role accepts a provider.
func (e RoleExtensibility) IsPluggable() bool { return e == RoleExtensibilityPluggable }

var unverifiedRoleIDs = map[string]struct{}{
	"pathfinder":   {},
	"cartographer": {},
	"jeweler":      {},
	"jewelcrafter": {},
}

// ValidateRoleReference rejects role names that were proposed but not approved
// for activation. Other custom role ids remain available to workspace-specific
// role registries.
func ValidateRoleReference(id string) error {
	normalized := strings.ToLower(strings.TrimSpace(id))
	if _, found := unverifiedRoleIDs[normalized]; found {
		return fmt.Errorf("role %q is not approved for activation", normalized)
	}
	return nil
}
