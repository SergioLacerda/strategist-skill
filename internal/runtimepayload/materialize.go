package runtimepayload

import (
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
// manifest and only then copies them into dest. Copying goes to a staging
// directory that replaces dest at the end, so a failure leaves dest untouched
// and nothing behind.
func Materialize(src fs.FS, m Manifest, goos, goarch, dest string) (Evidence, error) {
	comps, err := verifyComponents(src, m, goos, goarch)
	if err != nil {
		return Evidence{}, err
	}
	if err := activate(src, comps, dest); err != nil {
		return Evidence{}, err
	}
	return newEvidence(m, comps), nil
}

// verifyComponents selects the components for goos/goarch and verifies every
// directory tree in place against its pinned digest before anything is written.
func verifyComponents(src fs.FS, m Manifest, goos, goarch string) ([]Component, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	comps, err := m.selectFor(goos, goarch)
	if err != nil {
		return nil, err
	}
	for _, c := range comps {
		if err := verifyTree(src, c); err != nil {
			return nil, err
		}
	}
	return comps, nil
}

// activate copies into a staging directory and swaps it in for dest.
func activate(src fs.FS, comps []Component, dest string) error {
	staging := dest + ".staging"
	if err := os.RemoveAll(staging); err != nil {
		return fmt.Errorf("clear staging %s: %w", staging, err)
	}
	if err := extractAll(src, comps, staging); err != nil {
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

func extractAll(src fs.FS, comps []Component, staging string) error {
	budget := &budget{}
	for _, c := range comps {
		root := filepath.Join(staging, filepath.FromSlash(c.Dest))
		if err := os.MkdirAll(root, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", root, err)
		}
		if err := copyTree(src, c.File, root, budget); err != nil {
			return fmt.Errorf("extract %s: %w", c.File, err)
		}
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
