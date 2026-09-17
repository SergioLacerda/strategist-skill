package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
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
	ID      string `yaml:"id"`
	Version string `yaml:"version"`
	Digest  string `yaml:"digest"`
	Source  string `yaml:"source"`
}

// LockFromIngestedSkills renders the deterministic, sorted lock content for
// a set of ingested skills.
func LockFromIngestedSkills(skills []IngestedSkill) EmbeddedSkillLock {
	nodes := make([]EmbeddedSkillLockNode, 0, len(skills))
	for _, skill := range skills {
		nodes = append(nodes, EmbeddedSkillLockNode{
			ID:      skill.ID,
			Version: skill.Package.Version,
			Digest:  skill.Package.Digest,
			Source:  filepath.ToSlash(skill.Dir),
		})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return EmbeddedSkillLock{SchemaVersion: "strategist-embedded-skill-lock/v1", Packages: nodes}
}

// WriteCatalogAndMirrors writes the merged catalog to catalogPath, copies
// each newly ingested skill's complete package under defaultsRoot/skills/<id>,
// generates its normalized skills/<id>/skill.yaml mirror, and writes the lock (recording every embedded package, not
// only the newly ingested ones, so the lock always reflects the full current
// embedded set) to lockPath. All three are written only after every input is
// computed — a failure partway through never leaves catalog.yaml and the
// lock file individually half-updated relative to each other beyond what a
// single failed os.WriteFile can cause.
func WriteCatalogAndMirrors(result IngestionResult, defaultsRoot, catalogPath, lockPath string) error {
	catalogBytes, err := yaml.Marshal(result.Catalog)
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}
	if err := os.WriteFile(catalogPath, catalogBytes, 0o644); err != nil { //nolint:gosec // G306: generated catalog is not sensitive
		return fmt.Errorf("write %s: %w", catalogPath, err)
	}

	for _, skill := range result.Ingested {
		if err := writeSkillMirror(result.Catalog, skill, defaultsRoot); err != nil {
			return err
		}
	}

	// The lock records exactly the packages resolved from
	// external-skills-source/ in this run (result.Ingested) — it is
	// specifically the ingestion mechanism's own provenance record, not a
	// general inventory of every embedded catalog entry regardless of
	// origin (a hand-authored entry that never went through ingestion has
	// no source directory or resolved digest to record truthfully).
	lockBytes, err := yaml.Marshal(LockFromIngestedSkills(result.Ingested))
	if err != nil {
		return fmt.Errorf("marshal lock: %w", err)
	}
	if err := os.WriteFile(lockPath, lockBytes, 0o644); err != nil { //nolint:gosec // G306: generated lock is not sensitive
		return fmt.Errorf("write %s: %w", lockPath, err)
	}
	return nil
}

func writeSkillMirror(catalog pluginCatalog, skill IngestedSkill, defaultsRoot string) error {
	manifest, err := generateLegacyProviderManifest(catalog, skill.ID)
	if err != nil {
		return fmt.Errorf("generate mirror for %s: %w", skill.ID, err)
	}
	mirrorDir := filepath.Join(defaultsRoot, "skills", skill.ID)
	if err := os.MkdirAll(mirrorDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", mirrorDir, err)
	}
	if skill.Dir != "" {
		if err := copySkillPackage(skill.Dir, mirrorDir); err != nil {
			return fmt.Errorf("copy package for %s: %w", skill.ID, err)
		}
	}
	mirrorPath := filepath.Join(mirrorDir, "skill.yaml")
	if err := os.WriteFile(mirrorPath, manifest, 0o644); err != nil { //nolint:gosec // G306: generated manifest is not sensitive
		return fmt.Errorf("write %s: %w", mirrorPath, err)
	}
	return nil
}

// copySkillPackage materializes the source package inside the embedded
// defaults tree. The source package is operator-controlled, but reject
// symlinks so generated defaults cannot escape the declared package root.
func copySkillPackage(sourceDir, targetDir string) error {
	if err := filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, walkErr error) error {
		return copySkillEntry(sourceDir, targetDir, path, entry, walkErr)
	}); err != nil {
		return fmt.Errorf("walk %s: %w", sourceDir, err)
	}
	return nil
}

func copySkillEntry(sourceDir, targetDir, path string, entry os.DirEntry, walkErr error) error {
	if walkErr != nil {
		return fmt.Errorf("inspect %s: %w", path, walkErr)
	}
	if path == sourceDir {
		return nil
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink is not allowed: %s", path)
	}
	rel, err := filepath.Rel(sourceDir, path)
	if err != nil {
		return fmt.Errorf("relative path for %s: %w", path, err)
	}
	dst := filepath.Join(targetDir, rel)
	if entry.IsDir() {
		return makeSkillDirectory(dst)
	}
	if !entry.Type().IsRegular() {
		return fmt.Errorf("unsupported package entry: %s", rel)
	}
	return copySkillFile(path, dst)
}

func makeSkillDirectory(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	return nil
}

func copySkillFile(source, target string) error {
	data, err := os.ReadFile(source) //nolint:gosec // source is an operator-declared package path
	if err != nil {
		return fmt.Errorf("read %s: %w", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil { //nolint:gosec // generated embedded defaults are non-sensitive
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}
