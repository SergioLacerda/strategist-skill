package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/stretchr/testify/require"
)

func validProvider() Provider {
	return Provider{
		Enabled:             true,
		Endpoint:            "https://api.typesafe.ai/v1/systemone",
		CredentialRef:       "dotenv:.env#TYPESAFE_API_KEY",
		Model:               "jev-1.13.0",
		AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
		DataPolicy:          DataPolicy{AllowedFields: []string{"objective", "classification"}, MaxStateBytes: 8192},
	}
}

func validFile() File {
	return File{SchemaVersion: SchemaVersion, Providers: map[string]Provider{"jev": validProvider()}}
}

func TestSaveThenLoadRoundTripsAndIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	require.NoError(t, Save(path, validFile()))

	loaded, found, err := Load(path)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, validFile(), loaded)

	info, err := os.Stat(path)
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACLs are not represented by POSIX permission bits")
	}
	require.Zero(t, info.Mode().Perm()&0o077, "the file is operator-private")
}

func TestLoadMissingFileMeansNoIntegrationDecision(t *testing.T) {
	loaded, found, err := Load(filepath.Join(t.TempDir(), FileName))
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, loaded.Providers)
}

func TestSaveIsAtomicAndKeepsThePreviousFileOnInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	require.NoError(t, Save(path, validFile()))
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	broken := validFile()
	provider := broken.Providers["jev"]
	provider.Endpoint = "http://insecure.example.test"
	broken.Providers["jev"] = provider
	require.Error(t, Save(path, broken))

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after), "AC-17: a failed write preserves the previous valid state")
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temporary file is left behind")
}

func TestValidateRejectsUnsafeOrAmbiguousConfiguration(t *testing.T) {
	mutations := map[string]func(*Provider){
		"insecure endpoint":        func(p *Provider) { p.Endpoint = "http://api.example.test" },
		"literal secret":           func(p *Provider) { p.CredentialRef = "sk-live-literal-secret" },
		"unknown scheme":           func(p *Provider) { p.CredentialRef = "vault:path" },
		"missing model":            func(p *Provider) { p.Model = "" },
		"floating alias":           func(p *Provider) { p.Model = "jev-latest" },
		"unknown capability":       func(p *Provider) { p.AllowedCapabilities = []integration.Capability{"shell.exec"} },
		"no capability":            func(p *Provider) { p.AllowedCapabilities = nil },
		"no data policy":           func(p *Provider) { p.DataPolicy = DataPolicy{} },
		"non positive state limit": func(p *Provider) { p.DataPolicy.MaxStateBytes = 0 },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			file := validFile()
			provider := file.Providers["jev"]
			mutate(&provider)
			file.Providers["jev"] = provider
			err := file.Validate()
			require.Error(t, err)
			require.NotContains(t, err.Error(), "sk-live-literal-secret", "a rejected secret is never echoed")
		})
	}

	alias := validFile()
	provider := alias.Providers["jev"]
	provider.Model, provider.AllowModelAlias = "jev-latest", true
	alias.Providers["jev"] = provider
	require.NoError(t, alias.Validate(), "the alias is accepted only with an explicit opt-in")

	wrong := validFile()
	wrong.SchemaVersion = "strategist-integrations/v0"
	require.Error(t, wrong.Validate())
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	require.NoError(t, os.WriteFile(path, []byte("schema_version: "+SchemaVersion+"\nproviders: {}\nsurprise: 1\n"), 0o600))
	_, _, err := Load(path)
	require.Error(t, err)
}

func TestParityReportsBindingsThatDriftFromConfiguration(t *testing.T) {
	file := validFile()
	good := integration.Binding{Provider: "jev", Version: "1", Model: "jev-1.13.0", Consumer: "handoff", Capability: integration.CapHandoffValidate}

	require.Empty(t, Parity(file, []integration.Binding{good}))

	drift := map[string]integration.Binding{
		"undeclared provider":    {Provider: "ghost", Version: "1", Model: "m", Consumer: "handoff", Capability: integration.CapHandoffValidate},
		"model mismatch":         {Provider: "jev", Version: "1", Model: "jev-9.9.9", Consumer: "handoff", Capability: integration.CapHandoffValidate},
		"capability not allowed": {Provider: "jev", Version: "1", Model: "jev-1.13.0", Consumer: "handoff", Capability: integration.CapConfidenceEvaluate},
	}
	for name, binding := range drift {
		t.Run(name, func(t *testing.T) {
			require.NotEmpty(t, Parity(file, []integration.Binding{binding}))
		})
	}

	disabled := validFile()
	provider := disabled.Providers["jev"]
	provider.Enabled = false
	disabled.Providers["jev"] = provider
	require.NotEmpty(t, Parity(disabled, []integration.Binding{good}), "a binding to a disabled provider is drift")
}

func TestThresholdAndBudgetDefaultsAndValidation(t *testing.T) {
	provider := validProvider()
	require.InDelta(t, 0.90, provider.Threshold(), 1e-9)
	require.Equal(t, 2, provider.Budget())

	provider.ApprovalThreshold, provider.CallBudget = 0.95, 4
	require.InDelta(t, 0.95, provider.Threshold(), 1e-9)
	require.Equal(t, 4, provider.Budget())

	for name, mutate := range map[string]func(*Provider){
		"threshold above one": func(p *Provider) { p.ApprovalThreshold = 1.2 },
		"negative threshold":  func(p *Provider) { p.ApprovalThreshold = -0.1 },
		"negative budget":     func(p *Provider) { p.CallBudget = -1 },
	} {
		t.Run(name, func(t *testing.T) {
			file := validFile()
			bad := file.Providers["jev"]
			mutate(&bad)
			file.Providers["jev"] = bad
			require.Error(t, file.Validate())
		})
	}
}
