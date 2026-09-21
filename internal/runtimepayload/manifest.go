// Package runtimepayload verifies and materializes the pinned, private
// runtime (for example Node plus OpenSpec) a Ranked provider executes from.
// It is source-agnostic: the payload is read from an fs.FS so production can
// embed it per target while tests inject an in-memory one. Everything is
// fail-closed: nothing is written unless every needed component matches its
// pinned digest, and archive extraction rejects escapes and links.
package runtimepayload

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// SchemaVersion identifies the manifest format.
	SchemaVersion = "strategist-runtime-payload/v1"
	// AnyTarget marks a component that is independent of OS or architecture.
	AnyTarget = "any"

	// FormatTarGz is a gzip-compressed tar archive in the payload source.
	FormatTarGz = "tar.gz"
	// FormatZip is a zip archive in the payload source.
	FormatZip = "zip"
	// FormatDir is a directory tree in the payload source, verified by TreeDigest.
	FormatDir = "dir"

	// BuildInfoFile is the generated provenance file of a runtime tree; it is
	// neither hashed nor materialized.
	BuildInfoFile = "runtime.build.yaml"
)

var (
	// ErrTargetUnsupported means the manifest has no component for the current OS/arch.
	ErrTargetUnsupported = errors.New("runtime payload does not support this target")
	// ErrPayloadMissing means a pinned component file is absent from the payload source.
	ErrPayloadMissing = errors.New("runtime payload component is missing")
	// ErrDigestMismatch means a component differs from its pinned size or sha256.
	ErrDigestMismatch = errors.New("runtime payload digest mismatch")
	// ErrUnsafeArchive means an archive entry escapes the destination or is not a plain file.
	ErrUnsafeArchive = errors.New("runtime payload archive entry is unsafe")
)

// Manifest pins every component of a provider's private runtime.
type Manifest struct {
	SchemaVersion string      `yaml:"schema_version"`
	Provider      string      `yaml:"provider"`
	Launcher      Launcher    `yaml:"launcher"`
	Components    []Component `yaml:"components"`
}

// Launcher locates the executables inside the materialized runtime, as
// slash-separated paths relative to the runtime directory. Node is keyed by
// GOOS with a required "default".
type Launcher struct {
	Node   map[string]string `yaml:"node"`
	Script string            `yaml:"script"`
}

// Component is one pinned archive in the payload.
type Component struct {
	Name            string `yaml:"name"`
	Version         string `yaml:"version"`
	OS              string `yaml:"os"`
	Arch            string `yaml:"arch"`
	File            string `yaml:"file"`
	Format          string `yaml:"format"`
	SHA256          string `yaml:"sha256"`
	Size            int64  `yaml:"size"`
	Dest            string `yaml:"dest"`
	StripComponents int    `yaml:"strip_components,omitempty"`
}

// ParseManifest decodes and validates a manifest.
func ParseManifest(raw []byte) (Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse runtime payload manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// Marshal encodes the manifest.
func (m Manifest) Marshal() ([]byte, error) {
	out, err := yaml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime payload manifest: %w", err)
	}
	return out, nil
}

// Validate checks structure only; digests are verified against real bytes by Materialize.
func (m Manifest) Validate() error {
	if err := m.validateHeader(); err != nil {
		return err
	}
	if err := m.validateLauncher(); err != nil {
		return err
	}
	return m.validateComponents()
}

func (m Manifest) validateHeader() error {
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("runtime payload manifest: unsupported schema_version %q", m.SchemaVersion)
	}
	if m.Provider == "" || len(m.Components) == 0 {
		return errors.New("runtime payload manifest: provider and components are required")
	}
	return nil
}

func (m Manifest) validateLauncher() error {
	if m.Launcher.Node["default"] == "" || m.Launcher.Script == "" {
		return errors.New("runtime payload manifest: launcher needs node.default and script")
	}
	for _, rel := range append([]string{m.Launcher.Script}, mapValues(m.Launcher.Node)...) {
		if !safeRelative(rel) {
			return fmt.Errorf("runtime payload manifest: launcher path %q must be clean and relative", rel)
		}
	}
	return nil
}

func (m Manifest) validateComponents() error {
	for _, c := range m.Components {
		if err := c.validate(); err != nil {
			return fmt.Errorf("runtime payload manifest: component %q: %w", c.Name, err)
		}
	}
	return nil
}

func (c Component) validate() error {
	switch {
	case c.Name == "" || c.Version == "" || c.OS == "" || c.Arch == "":
		return errors.New("name, version, os and arch are required")
	case !fs.ValidPath(c.File) || c.File == ".":
		return fmt.Errorf("file %q must be a clean relative path", c.File)
	case c.Format != FormatTarGz && c.Format != FormatZip && c.Format != FormatDir:
		return fmt.Errorf("unsupported format %q", c.Format)
	case !validSHA256(c.SHA256):
		return errors.New("sha256 must be 64 hex characters")
	case c.Size <= 0:
		return errors.New("size must be positive")
	case !safeRelative(c.Dest):
		return fmt.Errorf("dest %q must be clean and relative", c.Dest)
	case c.StripComponents < 0:
		return errors.New("strip_components must not be negative")
	}
	return nil
}

// LauncherPaths returns the Node executable and script inside the runtime
// materialized at dest, and fails if either is absent.
func (m Manifest) LauncherPaths(dest, goos string) (node, script string, err error) {
	rel := m.Launcher.Node[goos]
	if rel == "" {
		rel = m.Launcher.Node["default"]
	}
	node = filepath.Join(dest, filepath.FromSlash(rel))
	script = filepath.Join(dest, filepath.FromSlash(m.Launcher.Script))
	for _, p := range []string{node, script} {
		if _, statErr := statFile(p); statErr != nil {
			return "", "", fmt.Errorf("%w: %s", ErrPayloadMissing, p)
		}
	}
	return node, script, nil
}

func safeRelative(p string) bool {
	return p != "" && p != "." && !strings.HasPrefix(p, "/") && !strings.Contains(p, `\`) && !strings.Contains(p, ":") && path.Clean(p) == p && !strings.HasPrefix(p, "../")
}

func validSHA256(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32
}

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
