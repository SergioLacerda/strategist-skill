package install

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

func withNormalizedSkillDigests(catalog pluginCatalog, skills []IngestedSkill) ([]IngestedSkill, error) {
	withDigests := append([]IngestedSkill(nil), skills...)
	for index := range withDigests {
		manifest, err := normalizedDigestManifest(catalog, withDigests[index].ID)
		if err != nil {
			return nil, fmt.Errorf("generate normalized package evidence for %s: %w", withDigests[index].ID, err)
		}
		digest, err := normalizedSkillDigest(withDigests[index].Dir, manifest)
		if err != nil {
			return nil, fmt.Errorf("digest normalized package %s: %w", withDigests[index].ID, err)
		}
		withDigests[index].NormalizedDigest = digest
	}
	return withDigests, nil
}

type normalizedSkillFile struct {
	path    string
	content []byte
}

func normalizedSkillDigest(sourceDir string, manifest []byte) (string, error) {
	files, err := readNormalizedSkillFiles(sourceDir)
	if err != nil {
		return "", err
	}
	files = append(files, normalizedSkillFile{path: "skill.yaml", content: manifest})
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	hash := sha256.New()
	for _, file := range files {
		_, _ = hash.Write([]byte(file.path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(file.content)
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

func readNormalizedSkillFiles(sourceDir string) ([]normalizedSkillFile, error) {
	files := make([]normalizedSkillFile, 0)
	if sourceDir == "" {
		return files, nil
	}
	err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		file, ok, err := normalizedSkillFileAt(sourceDir, path, entry, walkErr)
		if err != nil {
			return err
		}
		if ok {
			files = append(files, file)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", sourceDir, err)
	}
	return files, nil
}

func normalizedSkillFileAt(sourceDir, path string, entry fs.DirEntry, walkErr error) (normalizedSkillFile, bool, error) {
	if walkErr != nil {
		return normalizedSkillFile{}, false, fmt.Errorf("inspect %s: %w", path, walkErr)
	}
	if entry.IsDir() {
		return normalizedSkillFile{}, false, nil
	}
	rel, err := filepath.Rel(sourceDir, path)
	if err != nil {
		return normalizedSkillFile{}, false, fmt.Errorf("relative path for %s: %w", path, err)
	}
	rel = filepath.ToSlash(rel)
	if rel == "skill.yaml" {
		return normalizedSkillFile{}, false, nil
	}
	data, err := os.ReadFile(path) //nolint:gosec // source is an operator-declared package path
	if err != nil {
		return normalizedSkillFile{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	return normalizedSkillFile{path: rel, content: data}, true, nil
}
