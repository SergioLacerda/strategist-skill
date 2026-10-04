// Package config is the operator-owned integration configuration: which
// providers are enabled, where they live, how their credential is referenced
// and which data may leave. It stores a reference to a secret, never the secret.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"gopkg.in/yaml.v3"
)

const (
	// FileName is the integration configuration filename.
	FileName = "integrations.yaml"
	// SchemaVersion identifies the file format.
	SchemaVersion = "strategist-integrations/v1"
)

// DataPolicy bounds what the consumer may project into a provider request.
type DataPolicy struct {
	AllowedFields []string `yaml:"allowed_fields"`
	MaxStateBytes int      `yaml:"max_state_bytes"`
}

// Provider is the operator's decision about one provider.
type Provider struct {
	Enabled             bool                     `yaml:"enabled"`
	Endpoint            string                   `yaml:"endpoint,omitempty"`
	CredentialRef       string                   `yaml:"credential_ref,omitempty"`
	Model               string                   `yaml:"model,omitempty"`
	AllowModelAlias     bool                     `yaml:"allow_model_alias,omitempty"`
	AllowedCapabilities []integration.Capability `yaml:"allowed_capabilities,omitempty"`
	DataPolicy          DataPolicy               `yaml:"data_policy,omitempty"`
	// ApprovalThreshold is the confidence a consumer requires, strictly
	// exceeded, before it approves a delegated check. Zero means DefaultThreshold.
	ApprovalThreshold float64 `yaml:"approval_threshold,omitempty"`
	// CallBudget bounds provider calls per mission and consumer transition.
	// Zero means DefaultCallBudget. It is independent of the handoff attempts.
	CallBudget int `yaml:"call_budget,omitempty"`
}

const (
	// DefaultThreshold is the default approval threshold (strictly above 90%).
	DefaultThreshold = 0.90
	// DefaultCallBudget is the default number of provider calls per mission and transition.
	DefaultCallBudget = 2
)

// Threshold returns the effective approval threshold.
func (p Provider) Threshold() float64 {
	if p.ApprovalThreshold == 0 {
		return DefaultThreshold
	}
	return p.ApprovalThreshold
}

// Budget returns the effective call budget.
func (p Provider) Budget() int {
	if p.CallBudget == 0 {
		return DefaultCallBudget
	}
	return p.CallBudget
}

// File is the persisted configuration.
type File struct {
	SchemaVersion string              `yaml:"schema_version"`
	Providers     map[string]Provider `yaml:"providers"`
}

// Load reads the file. A missing file is "no decision recorded" (found false),
// not an error, so an upgrade of a workspace that never answered stays silent.
func Load(path string) (File, bool, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: operator configuration path
	if errors.Is(err, os.ErrNotExist) {
		return File{}, false, nil
	}
	if err != nil {
		return File{}, false, fmt.Errorf("read %s: %w", FileName, err)
	}
	var file File
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil && !errors.Is(err, io.EOF) {
		return File{}, false, fmt.Errorf("parse %s: %w", FileName, err)
	}
	if err := file.Validate(); err != nil {
		return File{}, false, err
	}
	return file, true, nil
}

// Save validates and atomically persists the file with owner-only permissions.
func Save(path string, file File) error {
	if file.SchemaVersion == "" {
		file.SchemaVersion = SchemaVersion
	}
	if err := file.Validate(); err != nil {
		return err
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", FileName, err)
	}
	return atomicWrite(path, data)
}
