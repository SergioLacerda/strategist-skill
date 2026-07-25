package dojo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	full := filepath.Join(runDir, fc.Path)
	exists := fileExists(full)
	items := []domain.DojoCheckItem{{
		Label:  fmt.Sprintf("files_created %s", fc.Path),
		Passed: exists,
		Detail: ifFail(exists, "file not found: "+full),
	}}
	if !exists {
		return items
	}
	content, err := os.ReadFile(full)
	if err != nil {
		return append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("files_created %s (read)", fc.Path),
			Passed: false,
			Detail: err.Error(),
		})
	}
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
		found := strings.Contains(text, section)
		items = append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("section %q in %s", section, fc.Path),
			Passed: found,
			Detail: ifFail(found, fmt.Sprintf("section %q not found", section)),
		})
	}
	return items
}

func checkMustContain(fc domain.DojoFileCheck, text string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	for _, needle := range fc.MustContain {
		found := strings.Contains(text, needle)
		items = append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("must_contain %q in %s", needle, fc.Path),
			Passed: found,
			Detail: ifFail(found, fmt.Sprintf("%q not found in file", needle)),
		})
	}
	return items
}

func checkMustNotContain(fc domain.DojoFileCheck, text string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	for _, needle := range fc.MustNotContain {
		found := strings.Contains(text, needle)
		items = append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("must_not_contain %q in %s", needle, fc.Path),
			Passed: !found,
			Detail: ifFail(!found, fmt.Sprintf("%q must not appear in file", needle)),
		})
	}
	return items
}
