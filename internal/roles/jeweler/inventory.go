package jeweler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// InventorySource declares one legacy artifact to capture. The path is kept
// relative in the resulting entry so the inventory is reproducible across
// workspaces; provenance remains an explicit caller-supplied fact.
type InventorySource struct {
	Key        string
	Kind       string
	SourcePath string
	Status     string
	Provenance string
}

// BuildWorkspaceInventory captures the legacy Treasure Chest files owned by a
// Strategist workspace. It intentionally inventories manifests and the
// compiled offline projection only; provider-side records remain owned by the
// external package and must be supplied as additional InventorySource values.
func BuildWorkspaceInventory(root string) ([]InventoryEntry, error) {
	return BuildInventory(root, []InventorySource{
		{Key: "treasure-chests", Kind: "governed_chests", SourcePath: ".strategist/treasure-chests.yaml", Status: "captured", Provenance: "workspace:legacy"},
		{Key: "jewels", Kind: "jewels", SourcePath: ".strategist/jewels.yaml", Status: "captured", Provenance: "workspace:legacy"},
		{Key: "potions", Kind: "potions", SourcePath: ".strategist/potions.yaml", Status: "captured", Provenance: "workspace:legacy"},
		{Key: "knowledge-index", Kind: "knowledge_index", SourcePath: ".strategist/knowledge.index.yaml", Status: "captured", Provenance: "workspace:legacy"},
		{Key: "compiled-index", Kind: "compiled_index", SourcePath: ".strategist/.compiled/.index.gz", Status: "captured", Provenance: "workspace:legacy"},
	})
}

// BuildInventory captures content digests without mutating the source tree.
// It fails closed for traversal, symlink, non-regular, missing, or duplicate
// entries so an inventory cannot accidentally certify data outside the legacy
// workspace or silently collapse two records into one.
func BuildInventory(root string, sources []InventorySource) ([]InventoryEntry, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("jeweler: inventory root is required")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("jeweler: inventory root: %w", err)
	}

	entries := make([]InventoryEntry, 0, len(sources))
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if _, ok := seen[source.Key]; ok {
			return nil, fmt.Errorf("jeweler: duplicate inventory source %q", source.Key)
		}
		seen[source.Key] = struct{}{}
		entry, err := inventoryEntry(root, resolvedRoot, source)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return entries, nil
}

func inventoryEntry(root, resolvedRoot string, source InventorySource) (InventoryEntry, error) {
	if err := validateInventorySource(source); err != nil {
		return InventoryEntry{}, err
	}
	relativePath, path, err := resolveInventoryFile(root, resolvedRoot, source.SourcePath)
	if err != nil {
		return InventoryEntry{}, err
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is constrained to the declared workspace root
	if err != nil {
		return InventoryEntry{}, fmt.Errorf("jeweler: read inventory %s: %w", relativePath, err)
	}
	sum := sha256.Sum256(data)
	return InventoryEntry{
		Key: source.Key, Kind: source.Kind, SourcePath: relativePath,
		Digest: "sha256:" + hex.EncodeToString(sum[:]), Status: source.Status, Provenance: source.Provenance,
	}, nil
}

func resolveInventoryFile(root, resolvedRoot, sourcePath string) (string, string, error) {
	relativePath, err := cleanInventoryPath(sourcePath)
	if err != nil {
		return "", "", err
	}
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	info, err := os.Lstat(path)
	if err != nil {
		return "", "", fmt.Errorf("jeweler: inventory %s: %w", relativePath, err)
	}
	if err := validateInventoryFile(info, relativePath); err != nil {
		return "", "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", "", fmt.Errorf("jeweler: resolve inventory %s: %w", relativePath, err)
	}
	if err := ensureInventoryWithinRoot(resolvedRoot, resolvedPath); err != nil {
		return "", "", fmt.Errorf("jeweler: inventory %s: %w", relativePath, err)
	}
	return relativePath, path, nil
}

func validateInventoryFile(info os.FileInfo, relativePath string) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("jeweler: inventory path must not be a symlink: %s", relativePath)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("jeweler: inventory path must be a regular file: %s", relativePath)
	}
	return nil
}

// InventoryDigest returns a stable digest for a validated inventory. Entry
// order is not meaningful, so the same canonical ordering used by migration
// manifests is applied before hashing.
func InventoryDigest(entries []InventoryEntry) (string, error) {
	canonical := append([]InventoryEntry(nil), entries...)
	if err := validateInventoryEntries(canonical); err != nil {
		return "", err
	}
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Key < canonical[j].Key })
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("jeweler: marshal inventory: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validateInventorySource(source InventorySource) error {
	for name, value := range map[string]string{
		"key": source.Key, "kind": source.Kind, "source_path": source.SourcePath,
		"status": source.Status, "provenance": source.Provenance,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("jeweler: inventory source %s is required", name)
		}
	}
	return nil
}

func validateInventoryEntries(entries []InventoryEntry) error {
	return validateEntries(entries, "inventory")
}

func validateEntries(entries []InventoryEntry, subject string) error {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Key == "" || entry.Kind == "" || entry.SourcePath == "" || entry.Digest == "" || entry.Status == "" || entry.Provenance == "" {
			return fmt.Errorf("jeweler: %s entries require key, kind, source_path, digest, status, and provenance", subject)
		}
		if _, ok := seen[entry.Key]; ok {
			return fmt.Errorf("jeweler: duplicate %s entry %q", subject, entry.Key)
		}
		seen[entry.Key] = struct{}{}
	}
	return nil
}

func cleanInventoryPath(path string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(path))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("jeweler: inventory source_path must stay within the workspace: %q", path)
	}
	return filepath.ToSlash(clean), nil
}

func ensureInventoryWithinRoot(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return fmt.Errorf("compare resolved paths: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("resolved path escapes the workspace")
	}
	return nil
}
