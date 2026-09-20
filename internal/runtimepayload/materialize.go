package runtimepayload

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

const (
	maxExtractedBytes = 2 << 30
	maxExtractedFiles = 100_000
)

// Evidence records what was verified and materialized, for ranked-runtimes.yaml.
type Evidence struct {
	Provider   string
	Components []ComponentEvidence
}

// ComponentEvidence identifies one verified component.
type ComponentEvidence struct {
	Name, Version, OS, Arch, SHA256 string
}

// Materialize verifies every component needed for goos/goarch against the
// manifest and only then extracts them into dest. Extraction goes to a staging
// directory that replaces dest at the end, so a failure leaves dest untouched
// and nothing behind.
func Materialize(src fs.FS, m Manifest, goos, goarch, dest string) (Evidence, error) {
	comps, blobs, err := verifyComponents(src, m, goos, goarch)
	if err != nil {
		return Evidence{}, err
	}
	if err := activate(src, comps, blobs, dest); err != nil {
		return Evidence{}, err
	}
	return newEvidence(m, comps), nil
}

func verifyComponents(src fs.FS, m Manifest, goos, goarch string) ([]Component, [][]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, nil, err
	}
	comps, err := m.selectFor(goos, goarch)
	if err != nil {
		return nil, nil, err
	}
	blobs := make([][]byte, len(comps))
	for i, c := range comps {
		if blobs[i], err = verifyComponent(src, c); err != nil {
			return nil, nil, err
		}
	}
	return comps, blobs, nil
}

// verifyComponent checks one component; directory trees are verified in place
// (no blob), archives are read and verified into memory.
func verifyComponent(src fs.FS, c Component) ([]byte, error) {
	if c.Format == FormatDir {
		return nil, verifyTree(src, c)
	}
	return readVerified(src, c)
}

// activate extracts into a staging directory and swaps it in for dest.
func activate(src fs.FS, comps []Component, blobs [][]byte, dest string) error {
	staging := dest + ".staging"
	if err := os.RemoveAll(staging); err != nil {
		return fmt.Errorf("clear staging %s: %w", staging, err)
	}
	if err := extractAll(src, comps, blobs, staging); err != nil {
		return cleanupStaging(staging, err)
	}
	if err := swapIn(staging, dest); err != nil {
		return cleanupStaging(staging, err)
	}
	return nil
}

func swapIn(staging, dest string) error {
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("replace %s: %w", dest, err)
	}
	if err := os.Rename(staging, dest); err != nil {
		return fmt.Errorf("activate runtime %s: %w", dest, err)
	}
	return nil
}

func newEvidence(m Manifest, comps []Component) Evidence {
	ev := Evidence{Provider: m.Provider}
	for _, c := range comps {
		ev.Components = append(ev.Components, ComponentEvidence{Name: c.Name, Version: c.Version, OS: c.OS, Arch: c.Arch, SHA256: c.SHA256})
	}
	return ev
}

// selectFor picks, per component name, the entry matching the target.
func (m Manifest) selectFor(goos, goarch string) ([]Component, error) {
	order, chosen := m.candidates(goos, goarch)
	out := make([]Component, 0, len(order))
	for _, name := range order {
		c, ok := chosen[name]
		if !ok {
			return nil, fmt.Errorf("%w: component %q has no build for %s/%s", ErrTargetUnsupported, name, goos, goarch)
		}
		out = append(out, c)
	}
	return out, nil
}

// candidates returns component names in first-seen order and, per name, the
// first entry that matches the target.
func (m Manifest) candidates(goos, goarch string) (order []string, chosen map[string]Component) {
	chosen = map[string]Component{}
	for _, c := range m.Components {
		if !slices.Contains(order, c.Name) {
			order = append(order, c.Name)
		}
		if _, taken := chosen[c.Name]; !taken && matches(c.OS, goos) && matches(c.Arch, goarch) {
			chosen[c.Name] = c
		}
	}
	return order, chosen
}

func matches(declared, actual string) bool { return declared == AnyTarget || declared == actual }

func readVerified(src fs.FS, c Component) ([]byte, error) {
	data, err := fs.ReadFile(src, c.File)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s (%s %s)", ErrPayloadMissing, c.File, c.Name, c.Version)
		}
		return nil, fmt.Errorf("read runtime payload %s: %w", c.File, err)
	}
	sum := sha256.Sum256(data)
	if int64(len(data)) != c.Size || hex.EncodeToString(sum[:]) != c.SHA256 {
		return nil, fmt.Errorf("%w: %s (%s %s)", ErrDigestMismatch, c.File, c.Name, c.Version)
	}
	return data, nil
}

func extractAll(src fs.FS, comps []Component, blobs [][]byte, staging string) error {
	budget := &budget{}
	for i, c := range comps {
		root := filepath.Join(staging, filepath.FromSlash(c.Dest))
		if err := os.MkdirAll(root, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", root, err)
		}
		if err := extractComponent(src, c, blobs[i], root, budget); err != nil {
			return fmt.Errorf("extract %s: %w", c.File, err)
		}
	}
	return nil
}

func extractComponent(src fs.FS, c Component, blob []byte, root string, b *budget) error {
	switch c.Format {
	case FormatTarGz:
		return extractTarGz(blob, root, c.StripComponents, b)
	case FormatZip:
		return extractZip(blob, root, c.StripComponents, b)
	case FormatDir:
		return copyTree(src, c.File, root, b)
	}
	return nil
}

type budget struct{ bytes, files int64 }

func (b *budget) file() error {
	b.files++
	if b.files > maxExtractedFiles {
		return fmt.Errorf("%w: too many files", ErrUnsafeArchive)
	}
	return nil
}

// cleanupStaging removes the staging tree after a failed materialization and
// keeps the original cause; a cleanup failure is joined, never swallowed.
func cleanupStaging(staging string, cause error) error {
	if rmErr := os.RemoveAll(staging); rmErr != nil {
		return errors.Join(cause, fmt.Errorf("clean staging %s: %w", staging, rmErr))
	}
	return cause
}
