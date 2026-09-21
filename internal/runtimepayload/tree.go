package runtimepayload

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// TreeDigest returns the sha256 of a directory tree in fsys (rooted at dir)
// and its payload byte size. The digest covers every regular file except
// BuildInfoFile, in sorted path order, as path NUL sha256(content) LF, so
// scripts/build-openspec-runtime.sh can compute the same value.
func TreeDigest(fsys fs.FS, dir string) (digest string, size int64, err error) {
	files, err := treeFiles(fsys, dir)
	if err != nil {
		return "", 0, err
	}
	h := sha256.New()
	for _, rel := range files {
		data, err := fs.ReadFile(fsys, path.Join(dir, rel))
		if err != nil {
			return "", 0, fmt.Errorf("read %s: %w", rel, err)
		}
		sum := sha256.Sum256(data)
		h.Write([]byte(rel + "\x00" + hex.EncodeToString(sum[:]) + "\n")) // hash.Hash.Write never returns an error
		size += int64(len(data))
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

// treeFiles lists regular files under dir (relative, slash-separated, sorted),
// excluding BuildInfoFile and rejecting anything that is not a plain file.
func treeFiles(fsys fs.FS, dir string) ([]string, error) {
	var files []string
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, walkErr error) error {
		rel, keep, err := treeFileEntry(dir, p, d, walkErr)
		if keep {
			files = append(files, rel)
		}
		return err
	})
	if err != nil {
		return nil, treeWalkError(dir, err)
	}
	sort.Strings(files)
	return files, nil
}

// treeFileEntry classifies one walked entry: keep is true for a plain file
// that belongs in the digest.
func treeFileEntry(dir, p string, d fs.DirEntry, walkErr error) (rel string, keep bool, err error) {
	switch {
	case walkErr != nil:
		return "", false, walkErr
	case d.IsDir():
		return "", false, nil
	case !d.Type().IsRegular():
		return "", false, notRegular(p)
	}
	rel = strings.TrimPrefix(p, dir+"/")
	return rel, rel != BuildInfoFile, nil
}

func treeWalkError(dir string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrPayloadMissing, dir)
	}
	return fmt.Errorf("walk %s: %w", dir, err)
}

func verifyTree(src fs.FS, c Component) error {
	sum, size, err := TreeDigest(src, c.File)
	if err != nil {
		return err
	}
	if sum != c.SHA256 || size != c.Size {
		return fmt.Errorf("%w: %s (%s %s)", ErrDigestMismatch, c.File, c.Name, c.Version)
	}
	return nil
}

// copyTree copies the verified directory tree into root.
func copyTree(src fs.FS, dir, root string, b *budget) error {
	files, err := treeFiles(src, dir)
	if err != nil {
		return err
	}
	for _, rel := range files {
		if err := copyTreeFile(src, dir, root, rel, b); err != nil {
			return err
		}
	}
	return nil
}

func copyTreeFile(src fs.FS, dir, root, rel string, b *budget) (err error) {
	f, err := src.Open(path.Join(dir, rel))
	if err != nil {
		return fmt.Errorf("open %s: %w", rel, err)
	}
	defer closeInto(f, rel, &err)
	return writeEntry(root, rel, 0, 0o644, f, b)
}
