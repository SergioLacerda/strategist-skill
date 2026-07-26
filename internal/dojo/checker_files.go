package dojo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// CheckFiles validates the files_created section of criteria against the run directory.
func CheckFiles(criteria domain.DojoCriteria, basePath string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	runDir := filepath.Join(basePath, criteria.RunDir)

	for _, fc := range criteria.FilesCreated {
		items = append(items, checkCreatedFile(runDir, fc)...)
	}
	return items
}

func checkCreatedFile(runDir string, fc domain.DojoFileCheck) []domain.DojoCheckItem {
	label := fmt.Sprintf("files_created %s", fc.Path)
	content, full, err := readFileUnderRoot(runDir, fc.Path)
	switch {
	case errors.Is(err, errPathEscape):
		return []domain.DojoCheckItem{newItem(label, false, err.Error())}
	case errors.Is(err, os.ErrNotExist):
		return []domain.DojoCheckItem{newItem(label, false, "file not found: "+full)}
	case err != nil:
		return []domain.DojoCheckItem{
			newItem(label, true, ""),
			newItem(fmt.Sprintf("files_created %s (read)", fc.Path), false, err.Error()),
		}
	}
	items := []domain.DojoCheckItem{newItem(label, true, "")}
	return append(items, checkFileContent(fc, string(content))...)
}

func checkFileContent(fc domain.DojoFileCheck, text string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	items = append(items, checkRequiredSections(fc, text)...)
	items = append(items, checkMustContain(fc, text)...)
	items = append(items, checkMustNotContain(fc, text)...)
	return items
}

func checkRequiredSections(fc domain.DojoFileCheck, text string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	for _, section := range fc.RequiredSections {
		label := fmt.Sprintf("section %q in %s", section, fc.Path)
		items = append(items, checkTextAssertion(label, text, section, true,
			fmt.Sprintf("section %q not found", section)))
	}
	return items
}

func checkMustContain(fc domain.DojoFileCheck, text string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	for _, needle := range fc.MustContain {
		label := fmt.Sprintf("must_contain %q in %s", needle, fc.Path)
		items = append(items, checkTextAssertion(label, text, needle, true,
			fmt.Sprintf("%q not found in file", needle)))
	}
	return items
}

func checkMustNotContain(fc domain.DojoFileCheck, text string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	for _, needle := range fc.MustNotContain {
		label := fmt.Sprintf("must_not_contain %q in %s", needle, fc.Path)
		items = append(items, checkTextAssertion(label, text, needle, false,
			fmt.Sprintf("%q must not appear in file", needle)))
	}
	return items
}
