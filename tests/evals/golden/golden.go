// Package golden provides deterministic snapshot comparison helpers for Strategist artifacts.
package golden

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"gopkg.in/yaml.v3"
)

var update = flag.Bool("update", false, "update golden files (disabled in CI)")

var (
	uuidPattern     = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b`)
	timePattern     = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})\b`)
	hashPattern     = regexp.MustCompile(`(?i)\b(?:sha256:)?[0-9a-f]{64}\b`)
	durationPattern = regexp.MustCompile(`\b\d+(?:\.\d+)?(?:ns|us|µs|ms|s|m|h)\b`)
	tempPathPattern = regexp.MustCompile(`(?i)(?:[A-Z]:\\|/)(?:[^\s"']*[\\/])?(?:tmp|temp)(?:[\\/][^\s"']*)?`)
)

// Mode selects how an artifact is canonicalized before comparison.
type Mode string

const (
	Exact      Mode = "exact"
	Normalized Mode = "normalized"
	Structural Mode = "structural"
)

// Assert compares actual with goldenPath, or updates the golden when -update is used locally.
func Assert(goldenPath string, actual []byte, mode Mode) error {
	canonical, err := Canonicalize(actual, mode)
	if err != nil {
		return err
	}
	if *update {
		if runningInCI() {
			return errors.New("golden: -update is forbidden in CI")
		}
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			return fmt.Errorf("golden: create directory: %w", err)
		}
		return os.WriteFile(goldenPath, canonical, 0o644)
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		return fmt.Errorf("golden: read %s: %w (run tests with -update to create it)", goldenPath, err)
	}
	if bytes.Equal(want, canonical) {
		return nil
	}
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A: difflib.SplitLines(string(want)), B: difflib.SplitLines(string(canonical)),
		FromFile: goldenPath, ToFile: "actual", Context: 3,
	})
	return fmt.Errorf("golden: artifact drift detected\n%s", diff)
}

// Canonicalize makes artifacts stable according to their comparison mode.
func Canonicalize(data []byte, mode Mode) ([]byte, error) {
	switch mode {
	case Exact:
		return ensureTrailingNewline(normalizeText(string(data))), nil
	case Normalized, Structural:
		var value any
		if err := yaml.Unmarshal(data, &value); err != nil {
			return nil, fmt.Errorf("golden: parse structured artifact: %w", err)
		}
		value = normalizeValue(value)
		if mode == Structural {
			value = structuralShape(value)
		}
		var buffer bytes.Buffer
		encoder := json.NewEncoder(&buffer)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(value); err != nil {
			return nil, fmt.Errorf("golden: encode canonical artifact: %w", err)
		}
		return buffer.Bytes(), nil
	default:
		return nil, fmt.Errorf("golden: unknown comparison mode %q", mode)
	}
}

func normalizeValue(value any) any {
	switch current := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(current))
		for key, child := range current {
			lower := strings.ToLower(key)
			switch {
			case strings.Contains(lower, "timestamp"), lower == "created_at", lower == "updated_at":
				out[key] = "<timestamp>"
			case strings.Contains(lower, "uuid") || strings.HasSuffix(lower, "_id") && uuidPattern.MatchString(fmt.Sprint(child)):
				out[key] = "<uuid>"
			case strings.Contains(lower, "duration"):
				out[key] = "<duration>"
			case lower == "hostname" || lower == "host":
				out[key] = "<hostname>"
			case strings.Contains(lower, "hash") || strings.Contains(lower, "checksum"):
				out[key] = "<hash>"
			case strings.Contains(lower, "path") && isVolatilePath(fmt.Sprint(child)):
				out[key] = "<temp-path>"
			default:
				out[key] = normalizeValue(child)
			}
		}
		return out
	case []any:
		out := make([]any, len(current))
		for index, child := range current {
			out[index] = normalizeValue(child)
		}
		return out
	case string:
		return normalizeText(current)
	default:
		return current
	}
}

func structuralShape(value any) any {
	switch current := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(current))
		for key, child := range current {
			out[key] = structuralShape(child)
		}
		return out
	case []any:
		shapes := make([]string, 0, len(current))
		unique := map[string]any{}
		for _, child := range current {
			shape := structuralShape(child)
			encoded, _ := json.Marshal(shape)
			key := string(encoded)
			if _, exists := unique[key]; !exists {
				unique[key] = shape
				shapes = append(shapes, key)
			}
		}
		sort.Strings(shapes)
		out := make([]any, 0, len(shapes))
		for _, key := range shapes {
			out = append(out, unique[key])
		}
		return out
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64, int, int64:
		return "number"
	default:
		return "string"
	}
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = timePattern.ReplaceAllString(text, "<timestamp>")
	text = uuidPattern.ReplaceAllString(text, "<uuid>")
	text = hashPattern.ReplaceAllString(text, "<hash>")
	text = durationPattern.ReplaceAllString(text, "<duration>")
	text = tempPathPattern.ReplaceAllString(text, "<temp-path>")
	return text
}

func isVolatilePath(path string) bool {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return filepath.IsAbs(path) || strings.Contains(lower, "/tmp/") || strings.Contains(lower, "/temp/")
}

func ensureTrailingNewline(text string) []byte {
	return []byte(strings.TrimRight(text, "\n") + "\n")
}

func runningInCI() bool {
	for _, name := range []string{"CI", "GITHUB_ACTIONS", "BUILD_BUILDID", "TF_BUILD"} {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" && !strings.EqualFold(value, "false") && value != "0" {
			return true
		}
	}
	return false
}
