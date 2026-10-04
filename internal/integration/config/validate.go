package config

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

const floatingAlias = "jev-latest"

var knownCapabilities = []integration.Capability{
	integration.CapHandoffValidate, integration.CapHandoffProject, integration.CapConfidenceEvaluate,
}

// Validate checks the file. Messages name the provider and the field, never a
// value, so a mistyped secret is not echoed back.
func (f File) Validate() error {
	if f.SchemaVersion != SchemaVersion {
		return fmt.Errorf("integrations: unsupported schema_version %q", f.SchemaVersion)
	}
	names := make([]string, 0, len(f.Providers))
	for name := range f.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	var errs []error
	for _, name := range names {
		if err := f.Providers[name].validate(); err != nil {
			errs = append(errs, fmt.Errorf("integrations: provider %s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

func (p Provider) validate() error {
	if err := p.validateFormats(); err != nil {
		return err
	}
	if !p.Enabled {
		return nil
	}
	return p.validateEnabled()
}

func (p Provider) validateFormats() error {
	if p.Endpoint != "" {
		if parsed, err := url.Parse(p.Endpoint); err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			return errors.New("endpoint must be a plain https URL")
		}
	}
	if p.CredentialRef != "" && !strings.HasPrefix(p.CredentialRef, "env:") && !strings.HasPrefix(p.CredentialRef, "dotenv:") {
		return errors.New("credential_ref must be an env: or dotenv: reference, never the secret")
	}
	return p.validateLimits()
}

func (p Provider) validateEnabled() error {
	switch {
	case p.Endpoint == "" || p.CredentialRef == "":
		return errors.New("an enabled provider needs endpoint and credential_ref")
	case p.Model == "":
		return errors.New("an enabled provider needs a pinned model")
	case p.Model == floatingAlias && !p.AllowModelAlias:
		return errors.New("the floating model alias needs allow_model_alias")
	case len(p.AllowedCapabilities) == 0:
		return errors.New("an enabled provider needs allowed_capabilities")
	case len(p.DataPolicy.AllowedFields) == 0 || p.DataPolicy.MaxStateBytes <= 0:
		return errors.New("an enabled provider needs a data_policy with fields and a positive size limit")
	}
	for _, capability := range p.AllowedCapabilities {
		if !slices.Contains(knownCapabilities, capability) {
			return errors.New("allowed_capabilities names an unknown capability")
		}
	}
	return nil
}

func (p Provider) validateLimits() error {
	if p.ApprovalThreshold < 0 || p.ApprovalThreshold > 1 || p.CallBudget < 0 {
		return errors.New("approval_threshold must be in (0,1] and call_budget cannot be negative")
	}
	return nil
}
