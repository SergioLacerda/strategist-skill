package check

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/conformance"
)

// evaluateRankedConformance builds a conformance record from the catalog's
// persisted digests and evaluates it without weakening the Ranked path.
func evaluateRankedConformance(root, slot string, stamp domain.CatalogRankedStamp) conformance.CertificationResult {
	record := conformance.CertificationRecord{
		SchemaVersion:   "strategist-conformance-record/v1",
		Role:            slot,
		Provider:        stamp.ID,
		Level:           conformance.Level(stamp.ConformanceLevel),
		PackageDigest:   stamp.CertificationDigest,
		AdapterDigest:   stamp.CertificationDigest,
		HostAPIDigest:   stamp.HostAPIDigest,
		ConnectorDigest: stamp.ConnectorDigest,
		TestSuiteDigest: stamp.TestSuiteDigest,
		PolicyDigest:    stamp.PolicyDigest,
		CertifiedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	input := conformance.CertificationInputDigests{
		Role:            slot,
		Provider:        stamp.ID,
		PackageDigest:   stamp.CertificationDigest,
		AdapterDigest:   stamp.CertificationDigest,
		HostAPIDigest:   liveHostAPIDigest(root, slot, stamp.HostAPIDigest),
		ConnectorDigest: stamp.ConnectorDigest,
		TestSuiteDigest: stamp.TestSuiteDigest,
		PolicyDigest:    stamp.PolicyDigest,
	}
	return conformance.EvaluateCertification(record, conformance.LevelC1Contract, input, time.Now())
}

// liveHostAPIDigest detects edits to the role contract materialized in the
// target workspace. Missing unrelated files fall back to the certified value.
func liveHostAPIDigest(root, slot, fallback string) string {
	roleMap, err := loadRoleSlotMap(root)
	if err != nil || roleMap[slot] == "" {
		return fallback
	}
	role := roleMap[slot]
	roleRaw, err := os.ReadFile(filepath.Join(root, "roles", role+".yaml")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		return fallback
	}
	skillRaw, err := os.ReadFile(filepath.Join(root, "internal_skills", role, "SKILL.md")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		return fallback
	}
	h := sha256.New()
	h.Write(roleRaw)
	h.Write(skillRaw)
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}
