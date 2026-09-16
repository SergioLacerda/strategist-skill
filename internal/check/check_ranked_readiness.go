package check

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/conformance"
)

// rankedCertificationReadiness reports Trust and PermissionGrant as Ready
// with a reason citing the catalog's certification digest, or Blocked when
// the catalog cannot be read, provider is not (or no longer) certified, or
// its conformance evidence fails evaluation (ADR-0043 DEC-006) — never
// silently falling back to Custom's trust.Verify/policy.EvaluateGrant path.
func rankedCertificationReadiness(root, slot, provider string) (trustCheck, grantCheck domain.ReadinessCheck) {
	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_catalog_unreadable", Detail: err.Error()}
		return blocked, blocked
	}
	stamp, ok, err := domain.FindCatalogRankedStamp(raw, provider)
	if err != nil {
		blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_catalog_invalid", Detail: err.Error()}
		return blocked, blocked
	}
	if !ok || !stamp.Certified() {
		blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_not_certified"}
		return blocked, blocked
	}
	if result := evaluateRankedConformance(root, slot, stamp); !result.Accepted {
		blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_conformance_rejected", Detail: strings.Join(result.ReasonCodes, ", ")}
		return blocked, blocked
	}
	ready := domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "ready_by_certification", Detail: stamp.CertificationDigest}
	return ready, ready
}

// evaluateRankedConformance builds a conformance.CertificationRecord from
// stamp's persisted digests (ADR-0043 DEC-006, generic over every Ranked
// (role, provider) pairing — no pairing-specific logic here) and evaluates
// it via conformance.EvaluateCertification. HostAPIDigest is additionally
// recomputed live from the target workspace's own roles/<role>.yaml +
// internal_skills/<role>/SKILL.md — these do materialize into
// .strategist/, unlike the connector/test-suite source, which are
// Strategist-tool internals compiled into the binary and never ship to a
// target workspace (see embedded_skill_conformance.go's own doc comments
// in the install package for why). A mismatch there means the role/skill
// file was edited locally since certification — genuine staleness.
// ConnectorDigest/TestSuiteDigest have no independently verifiable live
// counterpart outside a fresh Strategist build, so input intentionally
// mirrors record for those two dimensions: real, content-derived values,
// just not re-verifiable from inside a target workspace.
func evaluateRankedConformance(root, slot string, stamp domain.CatalogRankedStamp) conformance.CertificationResult {
	record := conformance.CertificationRecord{
		SchemaVersion:   "strategist-conformance-record/v1",
		Level:           conformance.Level(stamp.ConformanceLevel),
		PackageDigest:   stamp.CertificationDigest,
		AdapterDigest:   stamp.CertificationDigest,
		HostAPIDigest:   stamp.HostAPIDigest,
		ConnectorDigest: stamp.ConnectorDigest,
		TestSuiteDigest: stamp.TestSuiteDigest,
		CertifiedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	input := conformance.CertificationInputDigests{
		PackageDigest:   stamp.CertificationDigest,
		AdapterDigest:   stamp.CertificationDigest,
		HostAPIDigest:   liveHostAPIDigest(root, slot, stamp.HostAPIDigest),
		ConnectorDigest: stamp.ConnectorDigest,
		TestSuiteDigest: stamp.TestSuiteDigest,
	}
	return conformance.EvaluateCertification(record, conformance.LevelC1Contract, input, time.Now())
}

// liveHostAPIDigest recomputes the target workspace's current
// roles/<role>.yaml + internal_skills/<role>/SKILL.md digest, so a local
// edit since certification is detected as certification_stale by
// evaluateRankedConformance. Falls back to fallback (never flags stale)
// when the role for slot cannot be resolved or the files cannot be read —
// an unrelated lookup failure must not itself block an otherwise-valid
// Ranked binding.
func liveHostAPIDigest(root, slot, fallback string) string {
	roleMap, err := loadRoleSlotMap(root)
	if err != nil {
		return fallback
	}
	role := roleMap[slot]
	if role == "" {
		return fallback
	}
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
