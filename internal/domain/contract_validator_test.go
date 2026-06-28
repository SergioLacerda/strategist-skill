package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSlotWrite_AllowedPath(t *testing.T) {
	t.Parallel()
	scope := SlotWriteScope{
		SlotName:      "discovery",
		AllowedPrefix: ".analysis/pending/",
		AllowedExt:    ".md",
	}
	if err := ValidateSlotWrite(scope, ".analysis/pending/mission-123-analysis.md"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateSlotWrite_WrongPrefix(t *testing.T) {
	t.Parallel()
	scope := SlotWriteScope{
		SlotName:      "discovery",
		AllowedPrefix: ".analysis/pending/",
		AllowedExt:    ".md",
	}
	err := ValidateSlotWrite(scope, ".analysis/refined/mission-123-analysis.md")
	if err == nil {
		t.Fatal("expected error for path outside allowed prefix")
	}
	if !strings.Contains(err.Error(), "slot_write_scope_violation") {
		t.Errorf("error must contain 'slot_write_scope_violation', got: %v", err)
	}
}

func TestValidateSlotWrite_WrongExtension(t *testing.T) {
	t.Parallel()
	scope := SlotWriteScope{
		SlotName:      "discovery",
		AllowedPrefix: ".analysis/pending/",
		AllowedExt:    ".md",
	}
	err := ValidateSlotWrite(scope, ".analysis/pending/deploy.sh")
	if err == nil {
		t.Fatal("expected error for non-.md extension")
	}
	if !strings.Contains(err.Error(), "slot_write_scope_violation") {
		t.Errorf("error must contain 'slot_write_scope_violation', got: %v", err)
	}
}

func TestValidateSlotWrite_RangerScope(t *testing.T) {
	t.Parallel()
	rangerScope := SlotWriteScope{
		SlotName:      "discovery",
		AllowedPrefix: ".analysis/pending/",
		AllowedExt:    ".md",
	}
	// valid Ranger write
	if err := ValidateSlotWrite(rangerScope, ".analysis/pending/2026-06-23-my-mission-analysis.md"); err != nil {
		t.Errorf("valid ranger path should pass: %v", err)
	}
	// Ranger writing outside /pending/ — must be blocked
	err := ValidateSlotWrite(rangerScope, ".analysis/refined/2026-06-23-my-mission/analysis.md")
	if err == nil {
		t.Fatal("Ranger must not write to /refined/")
	}
	if !strings.Contains(err.Error(), "slot_write_scope_violation") {
		t.Errorf("error must contain 'slot_write_scope_violation', got: %v", err)
	}
}

func TestValidateSlotWrite_ArchivistScope(t *testing.T) {
	t.Parallel()
	archivistScope := SlotWriteScope{
		SlotName:      "refinement",
		AllowedPrefix: ".analysis/refined/",
		AllowedExt:    ".md",
	}
	// valid Archivist writes
	for _, path := range []string{
		".analysis/refined/mission-123/proposal.md",
		".analysis/refined/mission-123/design.md",
		".analysis/refined/mission-123/tasks.md",
		".analysis/refined/mission-123/analysis.md",
	} {
		if err := ValidateSlotWrite(archivistScope, path); err != nil {
			t.Errorf("valid archivist path %q should pass: %v", path, err)
		}
	}
	// Archivist writing outside .analysis/ — must be blocked
	err := ValidateSlotWrite(archivistScope, "SKILL.md")
	if err == nil {
		t.Fatal("Archivist must not write to repo root")
	}
}

func TestValidateArchivistPackage_AllFilesPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, f := range []string{"analysis.md", "proposal.md", "design.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := ValidateArchivistPackage(dir); err != nil {
		t.Errorf("expected nil for complete package, got: %v", err)
	}
}

func TestValidateArchivistPackage_MissingAnalysis(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, f := range []string{"proposal.md", "design.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	err := ValidateArchivistPackage(dir)
	if err == nil {
		t.Fatal("expected error when analysis.md is missing")
	}
	if !strings.Contains(err.Error(), "archivist_package_incomplete") {
		t.Errorf("error must contain 'archivist_package_incomplete', got: %v", err)
	}
	if !strings.Contains(err.Error(), "analysis.md") {
		t.Errorf("error must name the missing file 'analysis.md', got: %v", err)
	}
}

func TestValidateArchivistPackage_MissingMultipleFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := ValidateArchivistPackage(dir)
	if err == nil {
		t.Fatal("expected error when multiple files are missing")
	}
	if !strings.Contains(err.Error(), "archivist_package_incomplete") {
		t.Errorf("error must contain 'archivist_package_incomplete', got: %v", err)
	}
}

func TestValidateArchivistPackage_EmptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := ValidateArchivistPackage(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "archivist_package_incomplete") {
		t.Errorf("error must contain 'archivist_package_incomplete', got: %v", err)
	}
}

func TestValidateSlotWrite_AnyExtensionAllowed(t *testing.T) {
	t.Parallel()
	scope := SlotWriteScope{
		SlotName:      "execution",
		AllowedPrefix: ".analysis/archived/",
		AllowedExt:    "", // no extension restriction
	}
	if err := ValidateSlotWrite(scope, ".analysis/archived/report.md"); err != nil {
		t.Errorf("expected nil with no ext restriction: %v", err)
	}
	if err := ValidateSlotWrite(scope, ".analysis/archived/report.json"); err != nil {
		t.Errorf("expected nil with no ext restriction: %v", err)
	}
}
