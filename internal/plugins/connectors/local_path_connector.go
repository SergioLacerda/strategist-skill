package connectors

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
	"gopkg.in/yaml.v3"
)

// LocalPathConnector resolves ORKA-shaped skill packages (SKILL.md mandatory,
// references/, scripts/, templates/, assets/ optional — ADR-0033) from a
// filesystem directory into domain.PluginPackage values for the embedded-skill
// ingestion pipeline (.analysis/refined/20260913-embedded-skill-directory-catalog
// Task 1). It never claims invocation authority for the resolved package:
// like UnsupportedConnector (used today for external skill providers — see
// native_role_connector.go's own comment), actual invocation of an external
// skill's prompt content happens inside a separate process this CLI does not
// control, start, or observe.
type LocalPathConnector struct {
	ConnectorID         string
	ConnectorAPIVersion string
}

// Capabilities reports static resolve/probe ability only — no invoke, no
// remove, no enforcement observation, matching the honesty principle
// UnsupportedConnector already establishes for externally-invoked skills.
func (c LocalPathConnector) Capabilities(context.Context) RuntimeCapabilities {
	return RuntimeCapabilities{
		ConnectorID:  c.ConnectorID,
		ConnectorAPI: c.ConnectorAPIVersion,
		CanResolve:   true,
		CanProbe:     true,
	}
}

// Resolve reports whether locator.Path holds a structurally valid ORKA
// package, without activating or cataloguing it. ResolveLocalPackage is the
// function that actually produces the domain.PluginPackage the ingestion
// generator catalogues — this method only answers the RuntimeConnector
// contract's own resolve question.
func (c LocalPathConnector) Resolve(_ context.Context, locator RuntimeLocator) ConnectorResult {
	if locator.ID == "" || locator.Path == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "locator_incomplete"}
	}
	if _, err := ResolveLocalPackage(locator.Path); err != nil {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "local_package_invalid", Detail: err.Error()}
	}
	return ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "resolved_local_package", Detail: locator.Path}
}

// Probe validates static probe inputs without invoking external code.
func (c LocalPathConnector) Probe(_ context.Context, instance domain.InstalledInstance, entrypoint string) ConnectorResult {
	if instance.ID == "" || entrypoint == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "probe_input_incomplete"}
	}
	return ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "static_probe_ready"}
}

// Invoke never claims invocation authority — the resolved package's prompt
// content runs inside the host's own skill loader, a process this connector
// does not control.
func (c LocalPathConnector) Invoke(context.Context, InvocationEnvelope) ConnectorResult {
	return unsupported("invoke_not_claimed_by_local_path_connector")
}

// Remove never claims removal authority — this connector only resolves and
// reads; it never mutates the source directory it was pointed at.
func (c LocalPathConnector) Remove(context.Context, domain.InstalledInstance) ConnectorResult {
	return unsupported("remove_not_owned_by_local_path_connector")
}

// Observe reports enforcement as unsupported — matching every other
// connector variant that cannot verify runtime enforcement.
func (c LocalPathConnector) Observe(context.Context, domain.InstalledInstance) ObservationResult {
	return ObservationResult{
		ConnectorResult: unsupported("enforcement_unsupported"),
		Enforcement:     policy.EnforcementReport{ConnectorID: c.ConnectorID, Limitations: []string{"enforcement_not_supported"}},
	}
}

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

	digest, size, err := digestPackageDirectory(dir)
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
func digestPackageDirectory(dir string) (digest string, totalSize int64, err error) {
	type fileEntry struct {
		relPath string
		content []byte
	}
	var entries []fileEntry

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return fmt.Errorf("relative path for %q under %q: %w", path, dir, relErr)
		}
		if len(rel) > domain.MaxPluginPathLength {
			return fmt.Errorf("path %q exceeds %d characters", rel, domain.MaxPluginPathLength)
		}
		data, readErr := os.ReadFile(path) //nolint:gosec // G304: dir is an operator-declared ingestion source, not untrusted request input
		if readErr != nil {
			return fmt.Errorf("read %q: %w", path, readErr)
		}
		if len(data) > domain.MaxPluginManifestBytes {
			return fmt.Errorf("file %q exceeds %d bytes", rel, domain.MaxPluginManifestBytes)
		}
		entries = append(entries, fileEntry{relPath: filepath.ToSlash(rel), content: data})
		totalSize += int64(len(data))
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
