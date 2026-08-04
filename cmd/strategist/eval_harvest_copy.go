package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// harvestMission copies the default analysis.md plus every requested
// include type for one mission into destRoot/<mission_id>/, and returns how
// many fixture files were written. Re-running on the same mission
// overwrites the existing copy (DEC-4 — no versioning).
func harvestMission(basePath, destRoot, missionID string, includeTypes []string) (int, error) {
	srcDir, err := missionDir(basePath, missionID)
	if err != nil {
		return 0, err
	}
	destDir := filepath.Join(destRoot, missionID)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, fmt.Errorf("create %s: %w", destDir, err)
	}

	written := 0
	if err := copyHarvestFile(filepath.Join(srcDir, "analysis.md"), filepath.Join(destDir, "analysis.md")); err != nil {
		return written, err
	}
	written++

	for _, t := range includeTypes {
		src, dest := harvestIncludeSource(basePath, srcDir, destDir, missionID, t)
		if err := copyHarvestFile(src, dest); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// harvestIncludeSource resolves the source/destination pair for one
// --include type. "adr"/"report" live under <base_path>/archived/, never
// inside the mission directory (see 09-response.md § Artifact Contract).
func harvestIncludeSource(basePath, srcDir, destDir, missionID, includeType string) (src, dest string) {
	switch includeType {
	case "adr":
		return filepath.Join(basePath, "archived", missionID+"-adr.md"), filepath.Join(destDir, "adr.md")
	case "report":
		return filepath.Join(basePath, "archived", missionID+"-report.md"), filepath.Join(destDir, "report.md")
	default:
		filename := evalHarvestArtifactFiles[includeType]
		return filepath.Join(srcDir, filename), filepath.Join(destDir, filename)
	}
}

// missionDir resolves a mission's directory under refined/ or done/, in
// that order. treasure.ScannedMission carries no file path, so harvest
// resolves the directory itself.
func missionDir(basePath, missionID string) (string, error) {
	for _, sub := range []string{"refined", "done"} {
		candidate := filepath.Join(basePath, sub, missionID)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("mission %q not found under refined/ or done/", missionID)
}

func copyHarvestFile(src, dest string) error {
	in, err := os.Open(filepath.Clean(src)) //nolint:gosec // G304: harvest source paths are resolved from workspace mission directories under base_path, not raw user input
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	defer in.Close() //nolint:errcheck // read-only file; close error is not actionable

	out, err := os.Create(filepath.Clean(dest)) //nolint:gosec // G304: harvest destination is a computed path under tests/evals/regression/
	if err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	defer out.Close() //nolint:errcheck // best-effort close on error paths below; the happy-path close is checked explicitly

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dest, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", dest, err)
	}
	return nil
}
