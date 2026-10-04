package mechanisms

import (
	"errors"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Validate checks the catalog envelope and its family-aware row identities.
// The compatibility filename remains mechanisms.yaml, but each row belongs
// to an explicit canonical family and duplicate identity means family plus ID.
func (r Registry) Validate() error {
	if r.SchemaVersion != RegistrySchemaVersion {
		return fmt.Errorf("mechanisms registry: unsupported schema_version %q", r.SchemaVersion)
	}
	if len(r.Rows) == 0 {
		return errors.New("mechanisms registry: no rows")
	}
	return validateRegistryRows(r.Rows)
}

func validateRegistryRows(rows []Row) error {
	seen := map[domain.CanonicalIdentity]bool{}
	for _, row := range rows {
		if err := validateRow(row); err != nil {
			return fmt.Errorf("mechanisms registry: row %q: %w", row.ID, err)
		}
		identity, err := row.CanonicalIdentity()
		if err != nil {
			return fmt.Errorf("mechanisms registry: row %q: %w", row.ID, err)
		}
		if seen[identity] {
			return fmt.Errorf("mechanisms registry: duplicate identity %s/%s", identity.Family, identity.ID)
		}
		seen[identity] = true
	}
	return nil
}

// CanonicalIdentity returns the family-aware identity of a compatibility row.
func (row Row) CanonicalIdentity() (domain.CanonicalIdentity, error) {
	if !validFamilies[row.Family] {
		return domain.CanonicalIdentity{}, fmt.Errorf("unknown family %q", row.Family)
	}
	identity := domain.CanonicalIdentity{Family: domain.TaxonomyFamily(row.Family), ID: row.ID, Version: row.Version}
	if err := identity.Validate(); err != nil {
		return domain.CanonicalIdentity{}, fmt.Errorf("validate mechanism identity: %w", err)
	}
	return identity, nil
}
