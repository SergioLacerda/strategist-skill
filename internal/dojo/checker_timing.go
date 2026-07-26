package dojo

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// CheckTiming validates wall-time performance from a timing_criteria block.
// It reads total_wall_time_ms=<value> from the emit log.
// If timing_criteria is nil, or filesOnly is true, returns empty (no check performed) —
// timing depends on emit.log just like emit_log checks do.
func CheckTiming(criteria domain.DojoCriteria, logPath string, filesOnly bool) []domain.DojoCheckItem {
	if criteria.TimingCriteria == nil || filesOnly {
		return nil
	}
	tc := criteria.TimingCriteria

	raw, ok, items := readTimingLog(logPath)
	if !ok {
		return items
	}

	value, items := parseTotalWallTime(string(raw))
	if len(items) > 0 {
		return items
	}

	passed := value <= tc.MaxWallTimeMs
	return []domain.DojoCheckItem{{
		Label:  "timing total_wall_time_ms",
		Passed: passed,
		Detail: ifFail(passed, fmt.Sprintf("wall time %d ms exceeds max %d ms", value, tc.MaxWallTimeMs)),
	}}
}

func readTimingLog(logPath string) ([]byte, bool, []domain.DojoCheckItem) {
	if !fileExists(logPath) {
		return nil, false, []domain.DojoCheckItem{{
			Label:  "timing total_wall_time_ms",
			Passed: false,
			Detail: "emit.log not found — run the LLM scenario first",
		}}
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		return nil, false, []domain.DojoCheckItem{{
			Label:  "timing total_wall_time_ms",
			Passed: false,
			Detail: err.Error(),
		}}
	}
	return raw, true, nil
}

func parseTotalWallTime(log string) (int, []domain.DojoCheckItem) {
	const field = "total_wall_time_ms="
	idx := strings.Index(log, field)
	if idx < 0 {
		return 0, []domain.DojoCheckItem{{
			Label:  "timing total_wall_time_ms",
			Passed: false,
			Detail: "total_wall_time_ms not found in emit.log",
		}}
	}

	valStr := totalWallTimeValue(log[idx+len(field):])
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, []domain.DojoCheckItem{{
			Label:  "timing total_wall_time_ms",
			Passed: false,
			Detail: fmt.Sprintf("cannot parse total_wall_time_ms=%q: %v", valStr, err),
		}}
	}
	return val, nil
}

func totalWallTimeValue(rest string) string {
	end := strings.IndexAny(rest, " \t\n\r")
	if end < 0 {
		return rest
	}
	return rest[:end]
}
