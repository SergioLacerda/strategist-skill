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

// EmbeddedSkillLock is the on-disk shape of EmbeddedSkillLockFileName.
type EmbeddedSkillLock struct {
	SchemaVersion string                  `yaml:"schema_version"`
	Packages      []EmbeddedSkillLockNode `yaml:"packages"`
}

// EmbeddedSkillLockNode is one locked package entry: its id, version, content
// digest, and source directory at lock time.
type EmbeddedSkillLockNode struct {
	ID                 string `yaml:"id"`
	Version            string `yaml:"version"`
	Digest             string `yaml:"digest"`
	Source             string `yaml:"source"`
	ContractVersion    string `yaml:"contract_version"`
	OriginalDigest     string `yaml:"original_digest"`
	NormalizedDigest   string `yaml:"normalized_digest"`
	Transformation     string `yaml:"transformation"`
	LocalModifications string `yaml:"local_modifications"`
	VerificationState  string `yaml:"verification_state"`
	CanonicalSource    string `yaml:"canonical_source,omitempty"`
	UpstreamRevision   string `yaml:"upstream_revision,omitempty"`
	License            string `yaml:"license,omitempty"`
}

// LockFromIngestedSkills renders the deterministic, sorted lock content for
// a set of ingested skills.
func LockFromIngestedSkills(skills []IngestedSkill) EmbeddedSkillLock {
	nodes := make([]EmbeddedSkillLockNode, 0, len(skills))
	for _, skill := range skills {
		nodes = append(nodes, EmbeddedSkillLockNode{
			ID: skill.ID, Version: skill.Package.Version, Digest: skill.Package.Digest,
			Source: filepath.ToSlash(skill.Dir), ContractVersion: domain.CurrentSkillPackageContractVersion,
			OriginalDigest: skill.Package.Digest, NormalizedDigest: skill.Package.Digest,
			Transformation: "none", LocalModifications: "unknown", VerificationState: "declared",
			CanonicalSource: skill.Adapter.UpstreamRepo, UpstreamRevision: skill.Adapter.UpstreamCommit,
			License: skill.Adapter.License,
		})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return EmbeddedSkillLock{SchemaVersion: "strategist-embedded-skill-lock/v1", Packages: nodes}
}
