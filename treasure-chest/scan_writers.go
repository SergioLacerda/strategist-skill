package treasure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteScanOutputs regenerates cluster and gap output directories from scan results.
func WriteScanOutputs(clustersDir string, clusters []Cluster, gapsDir string, gaps []Gap) error {
	if err := RegenerateDir(clustersDir); err != nil {
		return err
	}
	if err := writeClusterFiles(clustersDir, clusters); err != nil {
		return err
	}
	if err := RegenerateDir(gapsDir); err != nil {
		return err
	}
	return writeGapFiles(gapsDir, gaps)
}

func writeClusterFiles(clustersDir string, clusters []Cluster) error {
	for _, c := range clusters {
		if err := WriteClusterFile(clustersDir, c); err != nil {
			return err
		}
	}
	return nil
}

func writeGapFiles(gapsDir string, gaps []Gap) error {
	for _, g := range gaps {
		if err := WriteGapFile(gapsDir, g); err != nil {
			return err
		}
	}
	return nil
}

// RegenerateDir recreates a generated output directory from scratch.
func RegenerateDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301
		return fmt.Errorf("create %s: %w", dir, err)
	}
	return nil
}

// WriteClusterFile writes one cluster artifact.
func WriteClusterFile(dir string, c Cluster) error {
	content := fmt.Sprintf(`---
id: %s
cited_missions: [%s]
tags: [%s]
trust: T2
generated_at: %s
---

# Cluster: %s

Recurring theme across: %s
`, c.ID, strings.Join(c.CitedMissions, ", "), strings.Join(c.Tags, ", "), c.GeneratedAt, c.ID, strings.Join(c.CitedMissions, ", "))
	path := filepath.Join(dir, c.ID+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // G306
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// WriteGapFile writes one gap artifact.
func WriteGapFile(dir string, g Gap) error {
	deps := "[]"
	if len(g.Dependencies) > 0 {
		deps = "[" + strings.Join(g.Dependencies, ", ") + "]"
	}
	content := fmt.Sprintf(`---
id: %s
cited_missions: [%s]
status: %s
dependencies: %s
trust: T2
generated_at: %s
---

# Gap: %s

Still pending in: %s
`, g.ID, strings.Join(g.CitedMissions, ", "), g.Status, deps, g.GeneratedAt, g.ID, strings.Join(g.CitedMissions, ", "))
	path := filepath.Join(dir, g.ID+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // G306
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
