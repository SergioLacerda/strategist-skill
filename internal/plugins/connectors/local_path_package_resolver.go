package connectors

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

const requiredSkillManifestFile = "SKILL.md"

// skillFrontmatter is the YAML frontmatter ORKA's SKILL.md carries (ADR-0033:
// "name", "description", "metadata: {version, author}").
type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Metadata    struct {
		Version string `yaml:"version"`
		Author  string `yaml:"author"`
	} `yaml:"metadata"`
}

// ResolveLocalPackage reads an ORKA-shaped skill package from dir and
// produces a domain.PluginPackage with a real content digest computed over
// every file in the package (SKILL.md plus whatever of references/, scripts/,
// templates/, assets/ are present), in sorted path order so the digest is
// reproducible. It enforces ADR-0033's declared limits
// (domain.MaxPluginManifestBytes for SKILL.md, domain.MaxPluginPathLength per
// path) and requires SKILL.md to exist with parseable name/version
// frontmatter. It never vendors or returns the package's prose content —
// only identity/digest metadata (ADR-0029 DEC-001: adapter-first, no blind
// vendoring).
func ResolveLocalPackage(dir string) (domain.PluginPackage, error) {
	return resolvePackage(dir, false)
}

// MaxEmbeddedRuntimeFileBytes caps one file of a build-generated runtime/
// subtree in an embedded skill package (a bundled CLI is far larger than the
// prose limit).
const MaxEmbeddedRuntimeFileBytes = 8 * 1024 * 1024

// embeddedRuntimeDir is the only subtree of an embedded skill package that may
// exceed domain.MaxPluginManifestBytes per file.
const embeddedRuntimeDir = "runtime/"

// ResolveEmbeddedPackage is ResolveLocalPackage for maintainer-controlled
// embedded ingestion: identical rules, except files under runtime/ (the
// prebuilt provider runtime produced by scripts/build-openspec-runtime.sh)
// may reach MaxEmbeddedRuntimeFileBytes. Custom client packages never get this
// allowance; they go through ResolveLocalPackage.
func ResolveEmbeddedPackage(dir string) (domain.PluginPackage, error) {
	return resolvePackage(dir, true)
}

func resolvePackage(dir string, allowRuntime bool) (domain.PluginPackage, error) {
	skillMDPath := filepath.Join(dir, requiredSkillManifestFile)
	skillMD, err := os.ReadFile(skillMDPath) //nolint:gosec // G304: dir is an operator-declared ingestion source, not untrusted request input
	if err != nil {
		return domain.PluginPackage{}, fmt.Errorf("local package %s: read %s: %w", dir, requiredSkillManifestFile, err)
	}
	if len(skillMD) > domain.MaxPluginManifestBytes {
		return domain.PluginPackage{}, fmt.Errorf("local package %s: %s exceeds %d bytes", dir, requiredSkillManifestFile, domain.MaxPluginManifestBytes)
	}

	frontmatter, err := parseSkillFrontmatter(skillMD)
	if err != nil {
		return domain.PluginPackage{}, fmt.Errorf("local package %s: %w", dir, err)
	}
	if frontmatter.Name == "" {
		return domain.PluginPackage{}, fmt.Errorf("local package %s: %s frontmatter missing required field %q", dir, requiredSkillManifestFile, "name")
	}

	digest, size, err := digestPackageDirectory(dir, allowRuntime)
	if err != nil {
		return domain.PluginPackage{}, err
	}

	return domain.PluginPackage{
		SchemaVersion:  "strategist-plugin-package/v1",
		ID:             frontmatter.Name,
		Publisher:      frontmatter.Metadata.Author,
		Version:        frontmatter.Metadata.Version,
		Digest:         digest,
		ArtifactURI:    "file://" + dir,
		ArtifactSize:   size,
		ManifestSchema: "orka/v1",
	}, nil
}

// parseSkillFrontmatter extracts the YAML frontmatter between the first pair
// of "---" delimiters in SKILL.md, per ORKA's convention.
func parseSkillFrontmatter(skillMD []byte) (skillFrontmatter, error) {
	const delim = "---"
	content := string(skillMD)
	if !hasPrefixTrimmed(content, delim) {
		return skillFrontmatter{}, fmt.Errorf("%s missing YAML frontmatter (must start with %q)", requiredSkillManifestFile, delim)
	}
	rest := content[len(delim):]
	end := indexOf(rest, "\n"+delim)
	if end < 0 {
		return skillFrontmatter{}, fmt.Errorf("%s frontmatter missing closing %q", requiredSkillManifestFile, delim)
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return skillFrontmatter{}, fmt.Errorf("%s frontmatter: %w", requiredSkillManifestFile, err)
	}
	return fm, nil
}

func hasPrefixTrimmed(s, prefix string) bool {
	for len(s) > 0 && (s[0] == '\n' || s[0] == ' ' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// digestPackageDirectory walks dir (SKILL.md plus any of references/,
// scripts/, templates/, assets/) in sorted relative-path order, enforcing
// domain.MaxPluginPathLength and domain.MaxPluginManifestBytes per file, and
// returns a stable sha256 over path+content pairs plus the total byte size.
func digestPackageDirectory(dir string, allowRuntime bool) (digest string, totalSize int64, err error) {
	var entries []fileEntry

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		entry, size, err := readPackageFile(dir, path, d, walkErr, allowRuntime)
		if err != nil || entry == nil {
			return err
		}
		entries = append(entries, *entry)
		totalSize += size
		return nil
	})
	if walkErr != nil {
		return "", 0, fmt.Errorf("local package %s: %w", dir, walkErr)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].relPath < entries[j].relPath })

	h := sha256.New()
	for _, entry := range entries {
		h.Write([]byte(entry.relPath))
		h.Write([]byte{0})
		h.Write(entry.content)
		h.Write([]byte{0})
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), totalSize, nil
}
