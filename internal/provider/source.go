// Package provider implements the local provider onboarding boundary.
package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	packageManifestName = "package.yaml"
	adapterManifestName = "adapter.yaml"
	legacyManifestName  = "skill.yaml"
	providerDirName     = "providers"
	transactionFileName = "provider-transactions.yaml"
)

// Source is a validated, already-materialized provider source.
type Source struct {
	Dir     string
	Package domain.PluginPackage
	Adapter domain.AdapterContract
	Legacy  *legacyView
	Files   map[string][]byte
}

// legacyView is deliberately a compatibility projection. It is read for
// conflict detection only; package, adapter, binding, and lock files remain
// the authorities for their respective facts.
type legacyView struct {
	ID             string   `yaml:"id"`
	CanonicalRole  string   `yaml:"canonical_role"`
	Roles          []string `yaml:"roles"`
	SupportedSlots []string `yaml:"supported_slots"`
	RiskScore      string   `yaml:"risk_score"`
}

func loadSource(input string) (Source, []Reason) {
	if reason := validateLocalSource(input); reason != nil {
		return Source{}, []Reason{*reason}
	}
	dir, err := filepath.Abs(input)
	if err != nil {
		return Source{}, []Reason{{Code: "source_path_invalid", Detail: fmt.Sprintf("resolve source: %v", err)}}
	}
	files, reasons := readSourceFiles(dir)
	if len(reasons) > 0 {
		return Source{Dir: dir, Files: files}, reasons
	}
	return decodeSource(dir, files)
}

// decodeSource strictly decodes the package and adapter manifests and, when
// present, the legacy compatibility view. Every failure is a Reason.
func decodeSource(dir string, files map[string][]byte) (Source, []Reason) {
	var reasons []Reason
	var pkg domain.PluginPackage
	if err := domain.DecodeStrictPluginYAML(files[packageManifestName], &pkg); err != nil {
		reasons = append(reasons, Reason{Code: "package_contract_invalid", Detail: err.Error()})
	}
	var adapter domain.AdapterContract
	if err := domain.DecodeStrictPluginYAML(files[adapterManifestName], &adapter); err != nil {
		reasons = append(reasons, Reason{Code: "adapter_contract_invalid", Detail: err.Error()})
	}
	view, reason := decodeLegacyView(files)
	if reason != nil {
		reasons = append(reasons, *reason)
	}
	return Source{Dir: dir, Package: pkg, Adapter: adapter, Legacy: view, Files: files}, reasons
}

func decodeLegacyView(files map[string][]byte) (*legacyView, *Reason) {
	raw, ok := files[legacyManifestName]
	if !ok {
		return nil, nil
	}
	var view legacyView
	if err := yaml.Unmarshal(raw, &view); err != nil {
		return nil, &Reason{Code: "legacy_view_invalid", Detail: err.Error()}
	}
	return &view, nil
}

func validateLocalSource(input string) *Reason {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return &Reason{Code: "source_required", Detail: "provider source is required"}
	}
	if isRemoteSource(trimmed) {
		return &Reason{Code: "remote_source_deferred", Detail: "v1 accepts only an already-materialized local directory"}
	}
	info, err := os.Stat(trimmed)
	if err != nil {
		return &Reason{Code: "source_unavailable", Detail: fmt.Sprintf("stat source: %v", err)}
	}
	if !info.IsDir() {
		return &Reason{Code: "source_not_directory", Detail: "provider source must be a directory"}
	}
	return nil
}

func isRemoteSource(source string) bool {
	if strings.HasPrefix(source, "git@") || strings.HasPrefix(source, "git+") {
		return true
	}
	if strings.Contains(source, "://") && !strings.HasPrefix(source, "file://") {
		return true
	}
	return false
}

func readSourceFiles(dir string) (map[string][]byte, []Reason) {
	files := map[string][]byte{}
	wanted := []string{packageManifestName, adapterManifestName}
	for _, name := range wanted {
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path) //nolint:gosec // source is explicitly operator-selected
		if err != nil {
			return files, []Reason{{Code: "manifest_missing", Detail: fmt.Sprintf("%s: %v", name, err)}}
		}
		files[name] = raw
	}
	if raw, err := os.ReadFile(filepath.Join(dir, legacyManifestName)); err == nil { //nolint:gosec // source is explicitly operator-selected
		files[legacyManifestName] = raw
	} else if !os.IsNotExist(err) {
		return files, []Reason{{Code: "legacy_view_unreadable", Detail: err.Error()}}
	}
	return files, nil
}
