package embeddedweapon

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
)

// legacyMirrorDir is the pre-ADR-0061 skills/<id>/ directory of an ingested id.
func legacyMirrorDir(defaultsRoot string, skill plugincatalog.IngestedSkill) string {
	return filepath.Join(defaultsRoot, names.SkillsDirName, skill.ID)
}

// removeLegacyMirrorDirs deletes the id-only mirror directories the versioned
// skills/<id>@<version>/ layout replaces. Only ids ingested in this run are
// touched, and a versioned directory is never mistaken for a legacy one.
func removeLegacyMirrorDirs(skills []plugincatalog.IngestedSkill, defaultsRoot string) error {
	for _, skill := range skills {
		dir := legacyMirrorDir(defaultsRoot, skill)
		if dir == filepath.Join(defaultsRoot, names.SkillsDirName, plugincatalog.SkillPayloadDirName(skill)) {
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove legacy mirror %s: %w", dir, err)
		}
	}
	return nil
}

// WriteSkillMirror writes one skill mirror directory under defaultsRoot.
func WriteSkillMirror(_ plugincatalog.PluginCatalog, skill plugincatalog.IngestedSkill, defaultsRoot string) error {
	mirrorDir := filepath.Join(defaultsRoot, names.SkillsDirName, plugincatalog.SkillPayloadDirName(skill))
	if err := os.MkdirAll(mirrorDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", mirrorDir, err)
	}
	if skill.Dir != "" {
		if err := CopySkillPackage(skill.Dir, mirrorDir); err != nil {
			return fmt.Errorf("copy package for %s: %w", skill.ID, err)
		}
	}
	// skill.yaml remains a normalized-digest input, but is not part of the
	// generation-2 embedded payload. A source package may contain the historic
	// file; remove it from the generated tree rather than shipping it again.
	if err := os.Remove(filepath.Join(mirrorDir, names.SkillYAMLName)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove compatibility view for %s: %w", skill.ID, err)
	}
	return nil
}

// CopySkillPackage materializes the source package inside the embedded
// defaults tree. The source package is operator-controlled, but reject
// symlinks so generated defaults cannot escape the declared package root.
func CopySkillPackage(sourceDir, targetDir string) error {
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
		return MakeSkillDirectory(dst)
	}
	if !entry.Type().IsRegular() {
		return fmt.Errorf("unsupported package entry: %s", rel)
	}
	return CopySkillFile(path, dst)
}

// MakeSkillDirectory creates path and any missing parents for a skill mirror.
func MakeSkillDirectory(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	return nil
}

// CopySkillFile copies the regular file source to target, creating the target directory.
func CopySkillFile(source, target string) error {
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
