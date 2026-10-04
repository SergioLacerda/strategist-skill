// Package catalog adapts materialized catalog files into domain contracts.
package catalog

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// FindRankedStamp decodes a catalog document, then delegates all taxonomy and
// certification decisions to the domain validator.
func FindRankedStamp(raw []byte, providerID string) (domain.CatalogRankedStamp, bool, error) {
	var document domain.CatalogRankedStampDocument
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return domain.CatalogRankedStamp{}, false, fmt.Errorf("parse catalog: %w", err)
	}
	stamp, found, err := domain.FindCatalogRankedStampInDocument(document, providerID)
	if err != nil {
		return domain.CatalogRankedStamp{}, false, fmt.Errorf("validate ranked stamp: %w", err)
	}
	return stamp, found, nil
}
