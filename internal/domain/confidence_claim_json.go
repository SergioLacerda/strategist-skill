package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// UnmarshalJSON accepts the pre-v1 `evidence` spelling only as a read-time
// compatibility alias. When both spellings are present they must carry the
// same IDs; canonical output always uses evidence_ids.
func (claim *ConfidenceClaim) UnmarshalJSON(data []byte) error {
	decoded, err := decodeConfidenceClaim(data)
	if err != nil {
		return err
	}
	if err := applyEvidenceCompatibility(data, &decoded); err != nil {
		return err
	}
	*claim = decoded
	return nil
}

func decodeConfidenceClaim(data []byte) (ConfidenceClaim, error) {
	type confidenceClaimAlias ConfidenceClaim
	var decoded confidenceClaimAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return ConfidenceClaim{}, fmt.Errorf("confidence claim: decode: %w", err)
	}
	return ConfidenceClaim(decoded), nil
}

func applyEvidenceCompatibility(data []byte, claim *ConfidenceClaim) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("confidence claim: inspect compatibility fields: %w", err)
	}
	canonical, hasCanonical := raw["evidence_ids"]
	legacy, hasLegacy := raw["evidence"]
	if !hasLegacy {
		return nil
	}
	legacyIDs, err := decodeEvidenceIDs(legacy, "evidence compatibility field")
	if err != nil {
		return err
	}
	if !hasCanonical {
		claim.EvidenceIDs = legacyIDs
		return nil
	}
	canonicalIDs, err := decodeEvidenceIDs(canonical, "evidence_ids field")
	if err != nil {
		return err
	}
	if !slices.Equal(canonicalIDs, legacyIDs) {
		return errors.New("confidence_invalid: evidence and evidence_ids contain different values")
	}
	return nil
}

func decodeEvidenceIDs(data json.RawMessage, field string) ([]string, error) {
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, fmt.Errorf("confidence_invalid: %s: %w", field, err)
	}
	return ids, nil
}
