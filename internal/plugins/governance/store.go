// Package governance owns workspace persistence for operator-controlled plugin
// trust policy and permission grants.
package governance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	// TrustPolicyFileName is the operator trust policy filename.
	TrustPolicyFileName = "trust-policy.yaml"
	// PermissionGrantsFileName is the persisted permission grant filename.
	PermissionGrantsFileName = "permission-grants.yaml"
)

// Load reads governance state from a workspace root. Missing files represent
// legacy/absent governance and are returned as zero values without error.
func Load(root string) (domain.TrustPolicy, domain.PermissionGrantFile, error) {
	policy, err := loadPolicy(root)
	if err != nil {
		return domain.TrustPolicy{}, domain.PermissionGrantFile{}, err
	}
	grants, err := loadGrants(root)
	if err != nil {
		return domain.TrustPolicy{}, domain.PermissionGrantFile{}, err
	}
	return policy, grants, nil
}

// Save atomically persists the supplied governance resources. Zero-valued
// resources are still valid and are useful when a caller intentionally
// initializes a new workspace policy boundary.
func Save(root string, policy domain.TrustPolicy, grants domain.PermissionGrantFile) error {
	if policy.SchemaVersion == "" {
		policy.SchemaVersion = domain.TrustPolicySchemaVersion
	}
	if grants.SchemaVersion == "" {
		grants.SchemaVersion = domain.PermissionGrantFileSchemaVersion
	}
	if err := validatePolicy(policy); err != nil {
		return err
	}
	if err := validateGrants(grants); err != nil {
		return err
	}
	if err := SavePolicy(root, policy); err != nil {
		return err
	}
	return SaveGrants(root, grants)
}

// SavePolicy atomically persists the operator trust policy without requiring
// permission grants to exist in the workspace yet.
func SavePolicy(root string, policy domain.TrustPolicy) error {
	if policy.SchemaVersion == "" {
		policy.SchemaVersion = domain.TrustPolicySchemaVersion
	}
	if err := validatePolicy(policy); err != nil {
		return err
	}
	data, err := yaml.Marshal(policy)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", TrustPolicyFileName, err)
	}
	if err := atomicWrite(filepath.Join(root, TrustPolicyFileName), data); err != nil {
		return fmt.Errorf("write %s: %w", TrustPolicyFileName, err)
	}
	return nil
}

// SaveGrants atomically persists permission grants without requiring a trust
// policy to exist in the workspace yet.
func SaveGrants(root string, grants domain.PermissionGrantFile) error {
	if grants.SchemaVersion == "" {
		grants.SchemaVersion = domain.PermissionGrantFileSchemaVersion
	}
	if err := validateGrants(grants); err != nil {
		return err
	}
	data, err := yaml.Marshal(grants)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", PermissionGrantsFileName, err)
	}
	if err := atomicWrite(filepath.Join(root, PermissionGrantsFileName), data); err != nil {
		return fmt.Errorf("write %s: %w", PermissionGrantsFileName, err)
	}
	return nil
}

// FindGrant returns the grant bound to both immutable digests.
func FindGrant(file domain.PermissionGrantFile, packageDigest, adapterDigest string) (domain.PermissionGrant, bool) {
	for _, grant := range file.Grants {
		if grant.PackageDigest == packageDigest && grant.AdapterDigest == adapterDigest {
			return grant, true
		}
	}
	return domain.PermissionGrant{}, false
}

//nolint:dupl // policy and grant files intentionally share the same strict YAML loading boundary.
func loadPolicy(root string) (domain.TrustPolicy, error) {
	data, err := os.ReadFile(filepath.Join(root, TrustPolicyFileName)) //nolint:gosec // fixed file below caller-owned root
	if errors.Is(err, os.ErrNotExist) {
		return domain.TrustPolicy{}, nil
	}
	if err != nil {
		return domain.TrustPolicy{}, fmt.Errorf("read %s: %w", TrustPolicyFileName, err)
	}
	var policy domain.TrustPolicy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return domain.TrustPolicy{}, fmt.Errorf("parse %s: %w", TrustPolicyFileName, err)
	}
	if err := validatePolicy(policy); err != nil {
		return domain.TrustPolicy{}, err
	}
	return policy, nil
}

//nolint:dupl // policy and grant files intentionally share the same strict YAML loading boundary.
func loadGrants(root string) (domain.PermissionGrantFile, error) {
	data, err := os.ReadFile(filepath.Join(root, PermissionGrantsFileName)) //nolint:gosec // fixed file below caller-owned root
	if errors.Is(err, os.ErrNotExist) {
		return domain.PermissionGrantFile{}, nil
	}
	if err != nil {
		return domain.PermissionGrantFile{}, fmt.Errorf("read %s: %w", PermissionGrantsFileName, err)
	}
	var grants domain.PermissionGrantFile
	if err := yaml.Unmarshal(data, &grants); err != nil {
		return domain.PermissionGrantFile{}, fmt.Errorf("parse %s: %w", PermissionGrantsFileName, err)
	}
	if err := validateGrants(grants); err != nil {
		return domain.PermissionGrantFile{}, err
	}
	return grants, nil
}

func validatePolicy(policy domain.TrustPolicy) error {
	if policy.SchemaVersion != domain.TrustPolicySchemaVersion {
		return fmt.Errorf("trust policy schema version invalid: %q", policy.SchemaVersion)
	}
	if policy.Revision == "" {
		return fmt.Errorf("trust policy revision is required")
	}
	return nil
}

func validateGrants(file domain.PermissionGrantFile) error {
	if file.SchemaVersion != domain.PermissionGrantFileSchemaVersion {
		return fmt.Errorf("permission grants schema version invalid: %q", file.SchemaVersion)
	}
	seen := make(map[string]struct{}, len(file.Grants))
	for _, grant := range file.Grants {
		if _, ok := seen[grant.ID]; ok {
			return fmt.Errorf("duplicate permission grant: %s", grant.ID)
		}
		seen[grant.ID] = struct{}{}
		if err := validateGrant(grant); err != nil {
			return err
		}
	}
	return nil
}

func validateGrant(grant domain.PermissionGrant) error {
	if grant.ID == "" || grant.PackageDigest == "" || grant.AdapterDigest == "" {
		return fmt.Errorf("permission grant identity is incomplete: %q", grant.ID)
	}
	for _, permission := range grant.GrantedPermissions {
		if !domain.IsKnownPluginPermission(permission) {
			return fmt.Errorf("permission grant %s contains unknown permission: %s", grant.ID, permission)
		}
	}
	return nil
}
