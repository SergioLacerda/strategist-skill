package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// The scan runs the whole HANDOFF integration flow with a synthetic credential
// that lives only in a .env file, then looks for it everywhere it must not be:
// the rendered output, the logs, every file the flow wrote, the process
// environment a child host would inherit and the command-line surface.

const rangerScanArtifact = `---
mission_id: m1
mission_status: ranger_done
sources_consulted: []
ranger_handoff_policy_facts:
  schema_version: strategist-ranger-handoff-policy-facts/v1
  require_recall: false
  require_boundary: false
  require_classification: false
  require_verdict: false
  informational_only: true
---

## mission_objective
Evaluate every refined package.
## known_facts
- id: F-1
  statement: a fact
## uncertainties
none
## recommended_refinement_focus
Preserve verdicts.
## confidence_summary
summary
## handoff
handoff
`

type scanRun struct {
	root      string
	workspace string
	stdout    string
	logs      string
}

// scanProvider answers like a hostile or sloppy provider: it can echo the
// credential it received into an error body or into a field of a 200 answer.
type scanProvider func(t *testing.T) http.HandlerFunc

func echoAndReject(status int) scanProvider {
	return func(*testing.T) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte("rejected " + r.Header.Get("Authorization")))
		}
	}
}

func approveWithEchoInModel(accept bool) scanProvider {
	return func(t *testing.T) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			body := sniperAnswer(t, "accept", 0.97)
			if !accept {
				body = bytes.Replace(body, []byte(`"jev-1.13.0"`), []byte(`"`+key+`"`), 1)
			}
			_, _ = w.Write(body)
		}
	}
}

// runScenario builds a workspace whose credential is only in .env, runs both
// lifecycle handoff evaluations against the provider and returns what it wrote.
func runScenario(t *testing.T, secret string, provider scanProvider, ranger bool) scanRun {
	t.Helper()
	return runScenarioWith(t, secret, provider, ranger, nil)
}

func runScenarioWith(t *testing.T, secret string, provider scanProvider, ranger bool, mutate func(*config.Provider)) scanRun {
	t.Helper()
	root := informationalMission(t)
	workspace := filepath.Dir(root)
	require.NoError(t, os.WriteFile(filepath.Join(workspace, ".env"), []byte("OTHER=1\nTYPESAFE_API_KEY="+secret+"\n"), 0o600))
	sim := jevtest.Start(t, provider(t))
	enableDotenvIntegration(t, root, sim, mutate)

	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	if ranger {
		pending := filepath.Join(workspace, ".analysis", "pending")
		require.NoError(t, os.MkdirAll(pending, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(pending, "m1-analysis.md"), []byte(rangerScanArtifact), 0o644))
		cmd := &cobra.Command{}
		var rangerOut bytes.Buffer
		cmd.SetOut(&rangerOut)
		require.NoError(t, runRangerHandoffEvaluate(cmd, rangerHandoffEvaluateOptions{Root: root, MissionID: "m1"}))
		out += rangerOut.String()
	}
	require.Equal(t, "Bearer "+secret, sim.LastAuthorization(), "the credential really was resolved from .env and used, so absence below is meaningful")
	require.FileExists(t, filepath.Join(root, "memory", "integration-calls.jsonl"), "the ledger exists and is part of the scan")
	return scanRun{root: root, workspace: workspace, stdout: out, logs: logs.String()}
}

func enableDotenvIntegration(t *testing.T, root string, sim *jevtest.Server, mutate func(*config.Provider)) {
	t.Helper()
	fields := append(handoffconsumer.Fields(handoff.TransitionArchivistToSniper), handoffconsumer.Fields(handoff.TransitionRangerToArchivist)...)
	provider := config.Provider{
		Enabled: true, Endpoint: sim.URL() + "/v1/systemone", CredentialRef: "dotenv:.env#TYPESAFE_API_KEY", Model: "jev-1.13.0",
		AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
		DataPolicy:          config.DataPolicy{AllowedFields: fields, MaxStateBytes: 16384},
	}
	if mutate != nil {
		mutate(&provider)
	}
	require.NoError(t, config.Save(filepath.Join(root, config.FileName), config.File{
		SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": provider},
	}))
	delegationRoundTripper = sim.Transport()
	t.Cleanup(func() { delegationRoundTripper = nil })
}

// filesContaining lists every file under dir whose bytes contain needle, except
// the .env file that is the credential's one legitimate home.
func filesContaining(t *testing.T, dir, needle string) []string {
	t.Helper()
	var found []string
	require.NoError(t, filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() == ".env" {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil && bytes.Contains(data, []byte(needle)) {
			found = append(found, path)
		}
		return nil
	}))
	return found
}

func assertSecretAbsent(t *testing.T, run scanRun, secret string) {
	t.Helper()
	require.NotContains(t, run.stdout, secret, "rendered output")
	require.NotContains(t, run.logs, secret, "logs")
	require.Empty(t, filesContaining(t, run.workspace, secret), "artifacts, lock, outcomes, ledger and memory")
}

func TestAC15CredentialNeverLeavesTheDotenvAcrossTheWholeFlow(t *testing.T) {
	const secret = "sk-scan-synthetic-0123456789"
	scenarios := map[string]scanProvider{
		"approved answer":                    approveWithEchoInModel(true),
		"401 echoing the bearer in the body": echoAndReject(http.StatusUnauthorized),
		"500 echoing the bearer in the body": echoAndReject(http.StatusInternalServerError),
		"200 with the key in the model name": approveWithEchoInModel(false),
	}
	for name, provider := range scenarios {
		t.Run(name, func(t *testing.T) {
			run := runScenario(t, secret, provider, true)
			assertSecretAbsent(t, run, secret)
			require.Contains(t, run.stdout, "delegation:", "the pre-check really ran")
		})
	}
}

func TestChildEnvironmentNeverInheritsTheDotenvCredential(t *testing.T) {
	const secret = "sk-scan-child-env-9876543210"
	runScenario(t, secret, approveWithEchoInModel(true), false)

	require.Empty(t, os.Getenv("TYPESAFE_API_KEY"), "the .env value is never exported to the process")
	for _, entry := range codexBridgeEnv(os.Environ(), t.TempDir()) {
		require.NotContains(t, entry, secret, "a host child built from os.Environ() cannot inherit the key")
	}
}

// credentialLikeFlag reports whether a flag name reads as a secret. A flag is
// credential-like when one of its dash-separated words says so; "tokens" (a count,
// as in --tokens-in) is not "token", and a "-ref" suffix names where a secret
// lives (env: or dotenv:), never the secret.
func credentialLikeFlag(name string) bool {
	if strings.HasSuffix(name, "-ref") {
		return false
	}
	words := map[string]bool{"token": true, "secret": true, "password": true, "passwd": true, "credential": true, "credentials": true, "apikey": true}
	parts := strings.Split(strings.ToLower(name), "-")
	for i, part := range parts {
		if words[part] || (part == "api" && i+1 < len(parts) && parts[i+1] == "key") {
			return true
		}
	}
	return false
}

func TestNoCommandAcceptsACredentialOnTheCommandLine(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		command.Flags().VisitAll(func(flag *pflag.Flag) {
			require.False(t, credentialLikeFlag(flag.Name), "%s --%s would put a secret in argv and shell history", command.CommandPath(), flag.Name)
		})
		for _, child := range command.Commands() {
			walk(child)
		}
	}
	walk(rootCmd)
}

func TestCredentialLikeFlagClassification(t *testing.T) {
	for _, name := range []string{"api-key", "token", "client-secret", "password"} {
		require.True(t, credentialLikeFlag(name), name)
	}
	for _, name := range []string{"tokens-in", "brief-tokens", "credential-ref", "mission-id"} {
		require.False(t, credentialLikeFlag(name), name)
	}
}

func configAndBindingFingerprint(t *testing.T, run scanRun) (string, string) {
	t.Helper()
	cfg, err := os.ReadFile(filepath.Join(run.root, config.FileName))
	require.NoError(t, err)
	outcome, err := handoff.NewOutcomeStore(run.root).Latest("m1")
	require.NoError(t, err)
	require.NotNil(t, outcome.Delegation, "the flow produced a delegation to compare")
	return fmt.Sprintf("%x", sha256.Sum256(cfg)), outcome.Delegation.BindingDigest
}

func TestAC14RotatingTheCredentialChangesNeitherConfigurationNorBinding(t *testing.T) {
	first := runScenario(t, "sk-rotation-first-0001", approveWithEchoInModel(true), false)
	second := runScenario(t, "sk-rotation-second-0002", approveWithEchoInModel(true), false)

	firstConfig, firstBinding := configAndBindingFingerprint(t, first)
	secondConfig, secondBinding := configAndBindingFingerprint(t, second)
	// The two runs use different simulator endpoints, so compare the binding
	// identity (provider, version, model, consumer, capability), which is endpoint-free.
	require.Equal(t, firstBinding, secondBinding, "AC-14: the contractual binding does not depend on the credential")
	require.NotEqual(t, firstConfig, secondConfig, "only the simulator URL differs between the two configuration files")
	assertSecretAbsent(t, first, "sk-rotation-first-0001")
	assertSecretAbsent(t, second, "sk-rotation-second-0002")
}

func TestTheScannerFindsAPlantedSecret(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested", "leak.txt"), []byte("token=sk-planted-secret"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("K=sk-planted-secret"), 0o600))

	found := filesContaining(t, dir, "sk-planted-secret")
	require.Equal(t, []string{filepath.Join(dir, "nested", "leak.txt")}, found, "a leak is found, and only .env is exempt")
}

func TestAC15AFloatingAliasCannotLaunderTheCredentialThroughTheModelName(t *testing.T) {
	const secret = "sk-alias-echo-5566778899"
	run := runScenarioWith(t, secret, approveWithEchoInModel(false), true, func(p *config.Provider) {
		p.Model, p.AllowModelAlias = "jev-latest", true
	})
	assertSecretAbsent(t, run, secret)
}
