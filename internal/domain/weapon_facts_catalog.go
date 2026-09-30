package domain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func readCatalogDocs(strategistRoot string) ([]weaponFactsDoc, error) {
	raw, err := os.ReadFile(filepath.Join(strategistRoot, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed path under the runtime root
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read plugin catalog: %w", err)
	}
	var file struct {
		Providers []weaponFactsDoc `yaml:"providers"`
	}
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse plugin catalog: %w", err)
	}
	return file.Providers, nil
}

// factsFromCatalog resolves provider, an id or an "id@version" reference, in
// the catalog. A plain id resolves only while exactly one version is catalogued.
// A reference whose id the catalog does not list is not this function's to
// answer (a Custom "name@version" package id), so it reports not found.
func factsFromCatalog(strategistRoot, provider string) (WeaponFacts, bool, error) {
	docs, err := readCatalogDocs(strategistRoot)
	if err != nil {
		return WeaponFacts{}, false, err
	}
	matches := catalogDocsForRef(docs, provider)
	switch len(matches) {
	case 0:
		return WeaponFacts{}, false, nil
	case 1:
		return matches[0].manifest(WeaponFactsSourceCatalog), true, nil
	}
	return WeaponFacts{}, false, ambiguousFactsError(provider, matches)
}

// catalogDocsForRef returns the catalog entries a reference names: every
// version of a plain id, or the one version of an "id@version" reference.
func catalogDocsForRef(docs []weaponFactsDoc, ref string) []weaponFactsDoc {
	id, version := ParseWeaponRef(ref)
	var matches []weaponFactsDoc
	for _, entry := range docs {
		if entry.ID == id && (version == "" || catalogDocVersion(entry) == version) {
			matches = append(matches, entry)
		}
	}
	return matches
}

func ambiguousFactsError(ref string, matches []weaponFactsDoc) error {
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, WeaponIdentity(match.ID, catalogDocVersion(match)))
	}
	return fmt.Errorf("%w: %q is catalogued as %s", ErrWeaponFactsAmbiguous, ref, strings.Join(names, ", "))
}

func catalogDocVersion(doc weaponFactsDoc) string {
	if doc.Version == "" {
		return DefaultWeaponVersion
	}
	return doc.Version
}

func factsFromCompatView(strategistRoot, provider string) (WeaponFacts, error) {
	raw, err := os.ReadFile(filepath.Join(strategistRoot, "skills", provider, "skill.yaml")) //nolint:gosec // G304: path derived from the runtime root and provider id
	if errors.Is(err, os.ErrNotExist) {
		return WeaponFacts{}, fmt.Errorf("%w: %s", ErrWeaponFactsNotFound, provider)
	}
	if err != nil {
		return WeaponFacts{}, fmt.Errorf("read compat view for %s: %w", provider, err)
	}
	return parseCompatView(provider, raw)
}

func parseCompatView(provider string, raw []byte) (WeaponFacts, error) {
	var doc weaponFactsDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return WeaponFacts{}, fmt.Errorf("parse compat view for %s: %w", provider, err)
	}
	doc.ID = provider
	return doc.manifest(WeaponFactsSourceCompatView), nil
}
