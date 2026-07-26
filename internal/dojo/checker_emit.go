package dojo

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// CheckEmitLog validates the emit_log section of criteria against a captured log file.
// logPath is the path to the emit.log written during an LLM run.
// If logPath does not exist and filesOnly is true, emit checks are skipped.
func CheckEmitLog(criteria domain.DojoCriteria, logPath string, filesOnly bool) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	if len(criteria.EmitLog.MustContain) == 0 && len(criteria.EmitLog.MustNotContain) == 0 {
		return items
	}

	if !fileExists(logPath) {
		return missingEmitLogItems(criteria, filesOnly)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		return []domain.DojoCheckItem{{
			Label:  "emit_log read",
			Passed: false,
			Detail: err.Error(),
		}}
	}
	log := string(raw)
	return checkEmitLogText(criteria, log)
}

func missingEmitLogItems(criteria domain.DojoCriteria, filesOnly bool) []domain.DojoCheckItem {
	if filesOnly {
		return nil
	}
	var items []domain.DojoCheckItem
	for _, key := range criteria.EmitLog.MustContain {
		items = append(items, newItem(fmt.Sprintf("emit %s", key), false,
			"emit.log not found — run the LLM scenario first"))
	}
	return items
}

func checkEmitLogText(criteria domain.DojoCriteria, log string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	for _, key := range criteria.EmitLog.MustContain {
		items = append(items, checkTextAssertion(fmt.Sprintf("emit %s", key), log, key, true,
			fmt.Sprintf("emit key %q not found in log", key)))
	}
	for _, key := range criteria.EmitLog.MustNotContain {
		items = append(items, checkTextAssertion(fmt.Sprintf("emit %s must NOT appear", key), log, key, false,
			fmt.Sprintf("emit key %q must not appear in log", key)))
	}
	return items
}
