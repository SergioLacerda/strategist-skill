package domain

import (
	"bytes"
	"fmt"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// DecodeStrictPluginYAML decodes bounded YAML and rejects unknown fields.
func DecodeStrictPluginYAML[T any](data []byte, out *T) error {
	if len(data) > MaxPluginManifestBytes {
		return fmt.Errorf("plugin yaml exceeds %d bytes", MaxPluginManifestBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(out); err != nil {
		if strings.Contains(err.Error(), "field") && strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("plugin yaml unknown field: %w", err)
		}
		return fmt.Errorf("plugin yaml decode: %w", err)
	}
	return nil
}

// ValidatePluginRelativePath rejects absolute, escaping, empty, or oversized paths.
func ValidatePluginRelativePath(path string) error {
	if path == "" {
		return fmt.Errorf("plugin path is required")
	}
	if len(path) > MaxPluginPathLength {
		return fmt.Errorf("plugin path exceeds %d characters", MaxPluginPathLength)
	}
	normalized := strings.ReplaceAll(path, "\\", "/")
	if filepath.IsAbs(path) || strings.HasPrefix(normalized, "/") {
		return fmt.Errorf("plugin path must be relative")
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return fmt.Errorf("plugin path must not escape the plugin root")
		}
	}
	clean := pathpkg.Clean(normalized)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("plugin path must not escape the plugin root")
	}
	return nil
}

func requireNonEmpty(errs *[]string, field, value string) {
	if value == "" {
		*errs = append(*errs, field+" is required")
	}
}

func requireDigest(errs *[]string, field, value string) {
	if value == "" {
		*errs = append(*errs, field+" is required")
		return
	}
	if !pluginDigestPattern.MatchString(value) {
		*errs = append(*errs, field+" must be sha256:<64 lowercase hex>")
	}
}

func joinPluginValidation(label string, errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s invalid: %s", label, strings.Join(errs, "; "))
}

func pluginAPIMatchesRange(version, expr string) bool {
	major, ok := pluginAPIMajor(version)
	if !ok {
		return false
	}
	for _, part := range strings.Fields(expr) {
		if !pluginAPIMatchesPart(major, part) {
			return false
		}
	}
	return expr != ""
}

func pluginAPIMatchesPart(major int, part string) bool {
	if len(part) < 2 {
		return false
	}
	switch {
	case strings.HasPrefix(part, ">="):
		minimum, err := strconv.Atoi(strings.TrimPrefix(part, ">="))
		return err == nil && major >= minimum
	case strings.HasPrefix(part, "<"):
		maximum, err := strconv.Atoi(strings.TrimPrefix(part, "<"))
		return err == nil && major < maximum
	case strings.HasPrefix(part, "="):
		exact, err := strconv.Atoi(strings.TrimPrefix(part, "="))
		return err == nil && major == exact
	default:
		return false
	}
}

func pluginAPIMajor(version string) (int, bool) {
	prefix := "strategist-plugin-api/"
	if !strings.HasPrefix(version, prefix) {
		return 0, false
	}
	major, err := strconv.Atoi(strings.TrimPrefix(version, prefix))
	if err != nil {
		return 0, false
	}
	return major, true
}
