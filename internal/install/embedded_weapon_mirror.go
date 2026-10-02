package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// legacyMirrorDir is the pre-ADR-0061 skills/<id>/ directory of an ingested id.
func legacyMirrorDir(defaultsRoot string, skill IngestedSkill) string {
	return filepath.Join(defaultsRoot, "skills", skill.ID)
}

// removeLegacyMirrorDirs deletes the id-only mirror directories the versioned
// skills/<id>@<version>/ layout replaces. Only ids ingested in this run are
// touched, and a versioned directory is never mistaken for a legacy one.
func removeLegacyMirrorDirs(skills []IngestedSkill, defaultsRoot string) error {
	for _, skill := range skills {
		dir := legacyMirrorDir(defaultsRoot, skill)
		if dir == filepath.Join(defaultsRoot, "skills", skillPayloadDirName(skill)) {
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove legacy mirror %s: %w", dir, err)
		}
	}
	return nil
}

func writeSkillMirror(catalog pluginCatalog, skill IngestedSkill, defaultsRoot string) error {
	manifest, err := skillDigestManifest(catalog, skill)
	if err != nil {
		return fmt.Errorf("generate mirror for %s: %w", skill.ID, err)
	}
	mirrorDir := filepath.Join(defaultsRoot, "skills", skillPayloadDirName(skill))
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
