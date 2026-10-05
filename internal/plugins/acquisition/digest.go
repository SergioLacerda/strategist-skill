package acquisition

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type treeEntry struct {
	Path string
	Mode uint32
	Kind byte
	Data []byte
}

func deterministicTreeDigest(root string) (string, []ManifestFile, error) {
	entries, err := collectTreeEntries(root)
	if err != nil {
		return "", nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return digestTreeEntries(entries), manifestFiles(entries), nil
}

func collectTreeEntries(root string) ([]treeEntry, error) {
	var entries []treeEntry
	err := filepath.WalkDir(root, func(path string, dirent fs.DirEntry, walkErr error) error {
		entry, err := collectTreeEntry(root, path, dirent, walkErr)
		if err != nil {
			return err
		}
		if entry != nil {
			entries = append(entries, *entry)
		}
		return nil
	})
	if err != nil {
		return nil, acquisitionError("digest_failed", err.Error())
	}
	return entries, nil
}

func collectTreeEntry(root, path string, dirent fs.DirEntry, walkErr error) (*treeEntry, error) {
	if walkErr != nil {
		return nil, fmt.Errorf("walk quarantine: %w", walkErr)
	}
	if path == root {
		return nil, nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, fmt.Errorf("relative path for %q: %w", path, err)
	}
	rel = filepath.ToSlash(rel)
	if rel == acquisitionManifestName || rel == readyMarkerName {
		return nil, nil
	}
	return treeEntryFromDirent(path, rel, dirent)
}

func treeEntryFromDirent(path, rel string, dirent fs.DirEntry) (*treeEntry, error) {
	if dirent.Type()&os.ModeSymlink != 0 {
		return nil, acquisitionError("unsafe_entry", "symbolic links are not allowed: "+rel)
	}
	info, err := dirent.Info()
	if err != nil {
		return nil, fmt.Errorf("inspect %q: %w", rel, err)
	}
	if info.IsDir() {
		return &treeEntry{Path: rel, Mode: uint32(info.Mode().Perm()), Kind: 'd'}, nil
	}
	if !info.Mode().IsRegular() {
		return nil, acquisitionError("unsafe_entry", "non-regular files are not allowed: "+rel)
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is contained by the transaction root
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", rel, err)
	}
	return &treeEntry{Path: rel, Mode: uint32(info.Mode().Perm()), Kind: 'f', Data: data}, nil
}

func digestTreeEntries(entries []treeEntry) string {
	h := sha256.New()
	for _, entry := range entries {
		_, _ = h.Write([]byte(entry.Path + "\x00" + string(entry.Kind) + "\x00" + strconv.FormatUint(uint64(entry.Mode), 8) + "\x00"))
		_, _ = h.Write(entry.Data)
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func manifestFiles(entries []treeEntry) []ManifestFile {
	manifestFiles := make([]ManifestFile, 0, len(entries))
	for _, entry := range entries {
		fileDigest := sha256.Sum256(entry.Data)
		manifestFiles = append(manifestFiles, ManifestFile{
			Path:   entry.Path,
			Mode:   fmt.Sprintf("%04o", entry.Mode),
			Kind:   string(entry.Kind),
			Bytes:  int64(len(entry.Data)),
			Digest: "sha256:" + hex.EncodeToString(fileDigest[:]),
		})
	}
	return manifestFiles
}

func cleanRelativeArchivePath(name string) (string, error) {
	if name == "" || strings.ContainsRune(name, '\x00') {
		return "", acquisitionError("archive_path_invalid", "archive entry has an empty or NUL-containing path")
	}
	name = strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(name, "/") {
		return "", acquisitionError("archive_path_invalid", "archive entry is absolute: "+name)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", acquisitionError("archive_path_invalid", "archive entry escapes quarantine: "+name)
	}
	return clean, nil
}
