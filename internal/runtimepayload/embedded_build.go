package runtimepayload

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// openSpecTreeGlob matches the embedded OpenSpec bundle of any catalogued
	// openspec-propose version (skills/<id>@<version>/runtime, ADR-0061 D11).
	openSpecTreeGlob = "skills/openspec-propose@*/runtime"
	embeddedProvider = "openspec-propose"
)

// resolveOpenSpecTreeDir returns the embedded OpenSpec bundle directory of one
// openspec-propose version. weaponVersion names it exactly; left empty it
// resolves only when a single version is embedded, and with several it refuses
// to pick one: no first-match and no "latest" (ADR-0060 Decision 7).
func resolveOpenSpecTreeDir(defaults fs.FS, weaponVersion string) (string, error) {
	if weaponVersion != "" {
		dir := "skills/" + embeddedProvider + "@" + weaponVersion + "/runtime"
		if _, err := fs.Stat(defaults, dir); err != nil {
			return "", fmt.Errorf("%w: %s", ErrPayloadMissing, dir)
		}
		return dir, nil
	}
	matches, err := fs.Glob(defaults, openSpecTreeGlob)
	if err != nil {
		return "", fmt.Errorf("locate embedded OpenSpec bundle: %w", err)
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %s", ErrPayloadMissing, openSpecTreeGlob)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("embedded OpenSpec bundle is ambiguous across versions %v; the bound Weapon version must select one", matches)
	}
}

type openSpecBuildInfo struct {
	Version    string `yaml:"version"`
	Bundle     string `yaml:"bundle"`
	TreeSHA256 string `yaml:"tree_sha256"`
	TreeBytes  int64  `yaml:"tree_bytes"`
}

// openSpecDest is the materialized bundle directory under weapon-runtime/<provider>.
const openSpecDest = "openspec"

// EmbeddedOpenSpecVersion reads the version certificate shipped with the
// embedded OpenSpec tree. It is independent of the host Node executable.
func EmbeddedOpenSpecVersion(defaults fs.FS) (string, error) {
	treeDir, err := resolveOpenSpecTreeDir(defaults, "")
	if err != nil {
		return "", err
	}
	var tree openSpecBuildInfo
	if err := readYAML(defaults, treeDir+"/"+BuildInfoFile, &tree); err != nil {
		return "", err
	}
	if strings.TrimSpace(tree.Version) == "" {
		return "", fmt.Errorf("embedded OpenSpec build info has no version")
	}
	return tree.Version, nil
}

// MaterializeOpenSpec installs the digest-verified OpenSpec bundle of the given
// openspec-propose Weapon version (empty when only one is embedded) and returns
// its private launcher path. Node is deliberately not part of this
// materialization; callers validate and record the host Node separately.
func MaterializeOpenSpec(defaults fs.FS, dest, weaponVersion string) (string, Evidence, error) {
	treeDir, err := resolveOpenSpecTreeDir(defaults, weaponVersion)
	if err != nil {
		return "", Evidence{}, err
	}
	var tree openSpecBuildInfo
	if err := readYAML(defaults, treeDir+"/"+BuildInfoFile, &tree); err != nil {
		return "", Evidence{}, err
	}
	script := openSpecDest + "/" + tree.Bundle
	manifest := Manifest{
		SchemaVersion: SchemaVersion,
		Provider:      embeddedProvider,
		Launcher:      Launcher{Node: map[string]string{"default": script}, Script: script},
		Components: []Component{{
			Name: "openspec", Version: tree.Version, OS: AnyTarget, Arch: AnyTarget,
			File: treeDir, Format: FormatDir, SHA256: tree.TreeSHA256,
			Size: tree.TreeBytes, Dest: openSpecDest,
		}},
	}
	evidence, err := Materialize(defaults, manifest, "any", "any", dest)
	if err != nil {
		return "", Evidence{}, err
	}
	return filepath.Join(dest, filepath.FromSlash(script)), evidence, nil
}

func readYAML(src fs.FS, name string, into any) error {
	raw, err := fs.ReadFile(src, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrPayloadMissing, name)
		}
		return fmt.Errorf("read %s: %w", name, err)
	}
	if err := yaml.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	return nil
}

// VerifyMaterializedOpenSpec recomputes the tree digest of the OpenSpec bundle
// materialized under runtimeDir (weapon-runtime/<provider>) and compares it
// with the digest recorded at install. It returns ErrPayloadMissing when the
// bundle directory is absent and ErrDigestMismatch when any file differs, was
// added or was removed.
func VerifyMaterializedOpenSpec(runtimeDir, wantSHA256 string) error {
	sum, _, err := TreeDigest(os.DirFS(runtimeDir), openSpecDest)
	if err != nil {
		return err
	}
	if wantSHA256 == "" || sum != wantSHA256 {
		return fmt.Errorf("%w: %s", ErrDigestMismatch, filepath.Join(runtimeDir, openSpecDest))
	}
	return nil
}
