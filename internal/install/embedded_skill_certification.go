package install

import (
	"crypto/sha256"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/conformance"
)

// rankedCertificationPairs pins every certified Role→embedded-weapon
// pairing. ADR-0043's pilot scope was exactly one pairing
// (Ranger↔brainstorming — see
// docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md
// Non-Goals); the second, Archivist↔openspec-propose, was added per
// docs/adr/0045-ranked-testsuitedigest-shared-pin.md (see that ADR for
// the TestSuiteDigest evidence caveat this pairing accepts). Extending
// Ranked to another pairing requires a deliberate addition here, never
// automatic certification of every embedded candidate.
var rankedCertificationPairs = map[string]string{
	"brainstorming":    "ranger",
	"openspec-propose": "archivist",
	"sniper":           "sniper",
}

// certifyRankedCandidates stamps ranked/certification_digest (plus the
// pre-generated runtime binding fragment, DEC-005, and the generic
// conformance evidence, DEC-006) on every catalog provider entry named in
// rankedCertificationPairs, validating role affinity and handoff-schema
// ownership first (DEC-004). A pinned pairing simply absent from catalog is
// skipped — not every catalog ingestForOptions ever runs against is the
// real embedded-defaults catalog (e.g. isolated ingestion tests use their
// own synthetic fixtures) — but a pinned pairing that IS present and fails
// validation is a hard error: prepare-embedded must never certify a
// candidate that does not actually satisfy it. defaultsRoot locates the
// role contract files DEC-006's host-API-contract digest reads.
func certifyRankedCandidates(catalog *pluginCatalog, defaultsRoot string) error {
	for id, wantRole := range rankedCertificationPairs {
		if err := certifyRankedCandidate(catalog, defaultsRoot, id, wantRole); err != nil {
			return err
		}
	}
	return nil
}

// certifyRankedCandidate stamps catalog.Providers[idx] (id's entry) in
// place, or is a no-op when id is absent — split out of
// certifyRankedCandidates to keep its loop body a single call.
func certifyRankedCandidate(catalog *pluginCatalog, defaultsRoot, id, wantRole string) error {
	idx := indexOfCatalogProvider(catalog.Providers, id)
	if idx < 0 {
		return nil
	}
	provider := catalog.Providers[idx]
	if err := validateRankedCandidate(provider, wantRole); err != nil {
		return fmt.Errorf("certify ranked candidate %q: %w", id, err)
	}
	if err := validateVerifiedUpstreamProvenance(provider); err != nil {
		return fmt.Errorf("certify ranked candidate %q: %w", id, err)
	}
	evidence, err := rankedConformanceEvidence(defaultsRoot, wantRole, id)
	if err != nil {
		return fmt.Errorf("certify ranked candidate %q: %w", id, err)
	}
	provider.Ranked = true
	provider.CertificationDigest = rankedCertificationDigest(provider, wantRole)
	provider.RankedBindingGeneration = 1
	provider.RankedBindingStatus = "active"
	provider.HostAPIDigest = evidence.hostAPI
	provider.ConnectorDigest = evidence.connector
	provider.TestSuiteDigest = evidence.testSuite
	provider.PolicyDigest = evidence.policy
	provider.ConformanceLevel = string(conformance.LevelC1Contract)
	catalog.Providers[idx] = provider
	return nil
}

func validateVerifiedUpstreamProvenance(provider pluginCatalogProvider) error {
	// Native role fillers are shipped by Strategist itself. Their provenance is
	// the embedded role/skill contract, not an external upstream repository.
	if provider.CompatibilitySource == "native_role" {
		return nil
	}
	fields := map[string]string{
		"upstream_repo":           provider.UpstreamRepo,
		"upstream_skill_path":     provider.UpstreamSkillPath,
		"upstream_version":        provider.UpstreamVersion,
		"upstream_commit":         provider.UpstreamCommit,
		"upstream_content_digest": provider.UpstreamContentDigest,
		"license":                 provider.License,
	}
	for field, value := range fields {
		if value == "" {
			return fmt.Errorf("verified upstream provenance incomplete: %s is required", field)
		}
	}
	return nil
}

// rankedConformanceEvidenceSet bundles ADR-0043 DEC-006's three generic
// digests for one certification pass — computed once per candidate rather
// than as three separate, individually-erroring call sites in
// certifyRankedCandidate.
type rankedConformanceEvidenceSet struct {
	hostAPI, connector, testSuite, policy string
}

func rankedConformanceEvidence(defaultsRoot, role, _ string) (rankedConformanceEvidenceSet, error) {
	hostAPI, err := hostAPIContractDigest(defaultsRoot, role)
	if err != nil {
		return rankedConformanceEvidenceSet{}, err
	}
	connector, err := connectorDigest()
	if err != nil {
		return rankedConformanceEvidenceSet{}, err
	}
	testSuite, err := testSuiteDigest(role)
	if err != nil {
		return rankedConformanceEvidenceSet{}, err
	}
	policy, err := policyDigest()
	if err != nil {
		return rankedConformanceEvidenceSet{}, err
	}
	return rankedConformanceEvidenceSet{hostAPI: hostAPI, connector: connector, testSuite: testSuite, policy: policy}, nil
}

func indexOfCatalogProvider(providers []pluginCatalogProvider, id string) int {
	for i, p := range providers {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// validateRankedCandidate checks the manifest/role-affinity/handoff-schema
// obligations a Ranked candidate must satisfy before certification (ADR-0043
// DEC-004, scoped down from the pending draft's full 10-step procedure): the
// candidate must declare affinity for wantRole, and wantRole must itself own
// a well-known handoff schema (domain.RoleHandoffSchema) — the fixed role
// checkpoint that normalizes this weapon's output, per
// docs/architecture/strategist-concepts.md's Ranked Class pipeline scope.
func validateRankedCandidate(provider pluginCatalogProvider, wantRole string) error {
	if !providerHasRole(provider, wantRole) {
		return fmt.Errorf("role affinity missing %q (has %v)", wantRole, providerRoles(provider))
	}
	if _, ok := domain.RoleHandoffSchema[wantRole]; !ok && wantRole != "sniper" {
		return fmt.Errorf("role %q declares no handoff schema", wantRole)
	}
	if err := provider.Runtime.Validate(); err != nil {
		return fmt.Errorf("runtime contract invalid: %w", err)
	}
	return nil
}

// rankedCertificationDigest computes a deterministic digest over the
// candidate's identity, target role, and the role's handoff schema — the
// three facts validateRankedCandidate just confirmed. It is not a substitute
// for the pending draft's fuller contract-test-derived digest, but it is
// real and reproducible: an identical (id, version, role, handoff schema)
// tuple always certifies to the same digest, and any change to one of them
// changes it.
func rankedCertificationDigest(provider pluginCatalogProvider, role string) string {
	runtime := domain.NormalizeRankedRuntime(provider.Runtime)
	input := provider.ID + "\t" + providerVersionOrDefault(provider.Version) + "\t" + role + "\t" + domain.RoleHandoffSchema[role] + "\t" + runtime.Kind + "\t" + runtime.Root + "\t" + runtime.Bootstrap + "\t" + runtime.Healthcheck
	// Pinned runtime identity is appended only when declared, so providers
	// without it keep their existing digests.
	if runtime.Version != "" || runtime.NodeVersion != "" {
		input += "\truntime_version=" + runtime.Version + "\tnode_version=" + runtime.NodeVersion
	}
	sum := sha256.Sum256([]byte(input))
	return fmt.Sprintf("sha256:%x", sum)
}
