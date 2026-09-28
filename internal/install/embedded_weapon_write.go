package install

import (
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// EmbeddedSkillLockFileName is the committed lock recording exactly which
// external-skills-source/ packages are embedded, at which digest — tasks.md
// Task 3.1. It lives at the repository root, sibling to external-skills-source/
// and go.mod, mirroring where go.sum/go.mod already live for the same reason
// (a committed, diffable record of exactly what a build consumes).
const EmbeddedSkillLockFileName = "external-skills-source.lock.yaml"

// embeddedSkillLockSchemaVersion is the only accepted lock schema. v2 records
// each package's Weapon origin and runtime kind; v1 locks are rejected.
const embeddedSkillLockSchemaVersion = "strategist-embedded-skill-lock/v2"

// EmbeddedSkillLock is the on-disk shape of EmbeddedSkillLockFileName.
type EmbeddedSkillLock struct {
	SchemaVersion string                  `yaml:"schema_version"`
	Packages      []EmbeddedSkillLockNode `yaml:"packages"`
}

// EmbeddedSkillLockNode is one locked package entry: its id, version, content
// digest, and source directory at lock time.
type EmbeddedSkillLockNode struct {
	ID                       string                    `yaml:"id"`
	Version                  string                    `yaml:"version"`
	Kind                     string                    `yaml:"kind"`
	Origin                   string                    `yaml:"origin"`
	RuntimeKind              string                    `yaml:"runtime_kind"`
	Digest                   string                    `yaml:"digest"`
	Source                   string                    `yaml:"source"`
	ContractVersion          string                    `yaml:"contract_version"`
	OriginalDigest           string                    `yaml:"original_digest"`
	NormalizedDigest         string                    `yaml:"normalized_digest"`
	Transformation           string                    `yaml:"transformation"`
	LocalModifications       string                    `yaml:"local_modifications"`
	VerificationState        string                    `yaml:"verification_state"`
	OriginalDigestEvidence   string                    `yaml:"original_digest_evidence"`
	NormalizedDigestEvidence string                    `yaml:"normalized_digest_evidence"`
	CanonicalSource          string                    `yaml:"canonical_source,omitempty"`
	UpstreamRevision         string                    `yaml:"upstream_revision,omitempty"`
	License                  string                    `yaml:"license,omitempty"`
	SupportedSlots           []string                  `yaml:"supported_slots,omitempty"`
	Composition              *domain.WeaponComposition `yaml:"composition,omitempty"`
}

// LockFromIngestedSkills renders the deterministic, sorted lock content for
// a set of ingested skills.
func LockFromIngestedSkills(skills []IngestedSkill) EmbeddedSkillLock {
	nodes := make([]EmbeddedSkillLockNode, 0, len(skills))
	for _, skill := range skills {
		normalizedDigest := skill.NormalizedDigest
		originalEvidence := "declared"
		normalizedEvidence := "declared"
		if normalizedDigest == "" {
			normalizedDigest = skill.Package.Digest
		} else {
			originalEvidence = "verified"
			normalizedEvidence = "verified"
		}
		nodes = append(nodes, EmbeddedSkillLockNode{
			ID: skill.ID, Version: skill.Package.Version, Kind: string(skill.Adapter.Kind), Origin: string(domain.WeaponOriginEmbedded), RuntimeKind: skill.Adapter.Runtime.Kind, Digest: skill.Package.Digest,
			Source: filepath.ToSlash(skill.Dir), ContractVersion: domain.CurrentSkillPackageContractVersion,
			OriginalDigest: skill.Package.Digest, NormalizedDigest: normalizedDigest,
			Transformation: "none", LocalModifications: "unknown", VerificationState: "declared",
			OriginalDigestEvidence: originalEvidence, NormalizedDigestEvidence: normalizedEvidence,
			CanonicalSource: skill.Adapter.UpstreamRepo, UpstreamRevision: skill.Adapter.UpstreamCommit,
			License:        skill.Adapter.License,
			SupportedSlots: skill.Adapter.SupportedSlots, Composition: skill.Adapter.Composition,
		})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return EmbeddedSkillLock{SchemaVersion: embeddedSkillLockSchemaVersion, Packages: nodes}
}
