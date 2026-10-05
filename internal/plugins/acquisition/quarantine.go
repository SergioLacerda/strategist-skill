package acquisition

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Acquire creates one disposable quarantine transaction. Any failure removes
// only the exact transaction created by this call; no managed repository path
// is touched.
func Acquire(ctx context.Context, req Request) (result Result, err error) {
	if err := validateRequest(req); err != nil {
		return Result{}, err
	}
	quarantineRoot, err := prepareQuarantineRoot(req.QuarantineRoot, req.ForbiddenRoots)
	if err != nil {
		return Result{}, err
	}
	transaction, err := createAcquisitionTransaction(quarantineRoot)
	if err != nil {
		return Result{}, err
	}
	defer removeFailedAcquisition(transaction, &err)

	return finishAcquisition(ctx, req, transaction)
}

func createAcquisitionTransaction(quarantineRoot string) (string, error) {
	transaction, err := os.MkdirTemp(quarantineRoot, "skill-for-hire-acquisition-")
	if err != nil {
		return "", acquisitionError("quarantine_create_failed", err.Error())
	}
	return transaction, nil
}

func removeFailedAcquisition(transaction string, resultErr *error) {
	if *resultErr == nil {
		return
	}
	if cleanupErr := os.RemoveAll(transaction); cleanupErr != nil {
		return
	}
}

func finishAcquisition(ctx context.Context, req Request, transaction string) (Result, error) {
	runner := req.Runner
	if runner == nil {
		runner = CommandGitRunner{}
	}
	archiveData, resolvedCommit, err := runner.FetchArchive(ctx, transaction, req.Repository, req.Commit, req.AllowNetwork)
	if err != nil {
		return Result{}, fmt.Errorf("fetch archive: %w", err)
	}
	if resolvedCommit != req.Commit {
		return Result{}, acquisitionError("commit_mismatch", fmt.Sprintf("requested %s, resolved %s", req.Commit, resolvedCommit))
	}
	if err := extractArchive(transaction, archiveData); err != nil {
		return Result{}, err
	}
	if err := validatePackageShape(transaction); err != nil {
		return Result{}, err
	}
	return finalizeAcquisition(req, transaction)
}

func finalizeAcquisition(req Request, transaction string) (Result, error) {
	digest, files, err := deterministicTreeDigest(transaction)
	if err != nil {
		return Result{}, err
	}
	manifest := newManifest(req, digest, files, time.Now())
	if err := validateManifest(manifest); err != nil {
		return Result{}, acquisitionError("manifest_invalid", err.Error())
	}
	if err := writeManifest(transaction, manifest); err != nil {
		return Result{}, err
	}
	if err := writeReady(transaction); err != nil {
		return Result{}, err
	}
	return Result{QuarantinePath: transaction, Manifest: manifest}, nil
}

func prepareQuarantineRoot(requested string, forbidden []string) (string, error) {
	root := requested
	if root == "" {
		root = os.TempDir()
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", acquisitionError("quarantine_root_invalid", err.Error())
	}
	absolute, err = canonicalPath(absolute)
	if err != nil {
		return "", acquisitionError("quarantine_root_invalid", err.Error())
	}
	if err := rejectUnderForbiddenRoot(absolute, forbidden); err != nil {
		return "", err
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return "", acquisitionError("quarantine_root_invalid", err.Error())
	}
	return absolute, nil
}

func rejectUnderForbiddenRoot(candidate string, forbidden []string) error {
	candidate = filepath.Clean(candidate)
	for _, root := range forbidden {
		if strings.TrimSpace(root) == "" {
			continue
		}
		if err := rejectUnderForbiddenRootEntry(candidate, root); err != nil {
			return err
		}
	}
	return nil
}

func rejectUnderForbiddenRootEntry(candidate, root string) error {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return acquisitionError("quarantine_root_invalid", err.Error())
	}
	absolute, err = canonicalPath(absolute)
	if err != nil {
		return acquisitionError("quarantine_root_invalid", err.Error())
	}
	rel, err := filepath.Rel(filepath.Clean(absolute), candidate)
	if err != nil {
		return acquisitionError("quarantine_root_invalid", err.Error())
	}
	if isWithinRoot(rel) {
		return acquisitionError("quarantine_root_forbidden", candidate+" is under "+absolute)
	}
	return nil
}

func isWithinRoot(relative string) bool {
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func canonicalPath(path string) (string, error) {
	path = filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved), nil
	}
	parent, err := canonicalPath(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

func validatePackageShape(root string) error {
	for _, name := range []string{"SKILL.md", "strategist.yaml"} {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err != nil {
			return acquisitionError("package_shape_invalid", "missing "+name)
		}
		if !info.Mode().IsRegular() {
			return acquisitionError("package_shape_invalid", name+" must be a regular file")
		}
		if info.Size() == 0 {
			return acquisitionError("package_shape_invalid", name+" is empty")
		}
	}
	return nil
}
