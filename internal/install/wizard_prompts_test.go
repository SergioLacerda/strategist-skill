package install

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout replaces os.Stdout with a pipe for the duration of fn and
// returns whatever was written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	fn()
	require.NoError(t, w.Close())
	os.Stdout = old
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

func TestCompatibleProviderOptionsPrefersDefaultCompatibleCandidate(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"},
		{
			ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist",
			CompatibilitySource: "embedded", Default: true,
			SupportedHandoffSchemas: []string{"schemas/handoff-archivist-to-sniper.schema.yaml"},
		},
	}}

	ids, defaultID, _, excluded := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml")
	assert.Equal(t, []string{"openspec-propose"}, ids)
	assert.Equal(t, "openspec-propose", defaultID)
	assert.Empty(t, excluded)
}

func TestCompatibleProviderOptionsIncludesAffiliatedCandidateWithoutNativeHandoff(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"},
		{
			ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist",
			CompatibilitySource: "embedded", Default: true,
			// The fixed Archivist role owns normalization into its handoff.
		},
	}}

	ids, defaultID, _, excluded := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml")
	assert.Equal(t, []string{"openspec-propose"}, ids)
	assert.Equal(t, "openspec-propose", defaultID)
	assert.Empty(t, excluded)
}

func TestCompatibleProviderOptionsReturnsEmptyWhenNoWeaponIsCompatible(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{}}

	ids, defaultID, _, excluded := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml")
	assert.Empty(t, ids)
	assert.Empty(t, defaultID)
	assert.Empty(t, excluded)
}

func TestCompatibleProviderOptionsReturnsRankedIDWhenCertified(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{
			ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger",
			CompatibilitySource: "embedded", Default: true,
			Ranked: true, CertificationDigest: "sha256:cert",
		},
		{ID: "openspec-explore", RiskScore: "write_analysis", CanonicalRole: "ranger", CompatibilitySource: "embedded"},
	}}

	ids, defaultID, rankedID, excluded := compatibleProviderOptions(catalog, "ranger", "")
	assert.ElementsMatch(t, []string{"brainstorming", "openspec-explore"}, ids)
	assert.Equal(t, "brainstorming", defaultID)
	assert.Equal(t, "brainstorming", rankedID)
	assert.Empty(t, excluded)
}

func TestCompatibleProviderOptionsRankedIDEmptyWhenNoCandidateCertified(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger", CompatibilitySource: "embedded", Default: true},
	}}

	_, _, rankedID, _ := compatibleProviderOptions(catalog, "ranger", "")
	assert.Empty(t, rankedID)
}

func TestWithRankedOption_PrependsSynthenticEntryWhenRankedIDPresent(t *testing.T) {
	t.Parallel()

	options, uiDefault := withRankedOption([]string{"brainstorming", "openspec-explore"}, "brainstorming", "brainstorming")
	assert.Equal(t, []string{"brainstorming::ranked", "brainstorming", "openspec-explore"}, options)
	assert.Equal(t, "brainstorming::ranked", uiDefault)
}

func TestWithRankedOption_PassthroughWhenNoRankedID(t *testing.T) {
	t.Parallel()

	options, uiDefault := withRankedOption([]string{"brainstorming", "openspec-explore"}, "brainstorming", "")
	assert.Equal(t, []string{"brainstorming", "openspec-explore"}, options)
	assert.Equal(t, "brainstorming", uiDefault)
}

func TestSplitRankedChoice(t *testing.T) {
	t.Parallel()

	provider, mode := splitRankedChoice("brainstorming::ranked", "brainstorming")
	assert.Equal(t, "brainstorming", provider)
	assert.Equal(t, "ranked", mode)

	provider, mode = splitRankedChoice("brainstorming", "brainstorming")
	assert.Equal(t, "brainstorming", provider)
	assert.Equal(t, "custom", mode)

	provider, mode = splitRankedChoice("openspec-explore", "brainstorming")
	assert.Equal(t, "openspec-explore", provider)
	assert.Equal(t, "custom", mode)

	provider, mode = splitRankedChoice("custom-skill", "")
	assert.Equal(t, "custom-skill", provider)
	assert.Equal(t, "custom", mode)
}

func TestPromptExecutionSlot_ResolvesRankedAndCustomModes(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "sniper", RiskScore: "controlled", CompatibilitySource: "native_role", Ranked: true, CertificationDigest: "sha256:cert"},
	}}
	b := i18n.BundleFor("en")

	provider, mode, err := promptExecutionSlot(NewTextPrompter(strings.NewReader("sniper::ranked\n")), b, catalog, knownProviderRisk)
	require.NoError(t, err)
	assert.Equal(t, "sniper", provider)
	assert.Equal(t, domain.SlotBindingModeRanked, mode)

	provider, mode, err = promptExecutionSlot(NewTextPrompter(strings.NewReader("custom-execution\n")), b, catalog, knownProviderRisk)
	require.NoError(t, err)
	assert.Equal(t, "custom-execution", provider)
	assert.Equal(t, domain.SlotBindingModeCustom, mode)
}

func TestPrintExcludedCandidatesPrintsEachIDWithItsReasons(t *testing.T) {
	// No t.Parallel(): captureStdout swaps the process-global os.Stdout, which
	// is unsafe to do concurrently with other tests.
	excluded := []excludedProviderOption{
		{
			id: "sdd-ask",
			reasons: []domain.CompatibilityReason{
				{Dimension: "role_affinity", Code: "role_mismatch", Detail: "provider declares roles [sniper], role contract is \"archivist\""},
			},
		},
		{
			id: "batata",
			reasons: []domain.CompatibilityReason{
				{Dimension: "handoff_schema", Code: "unsupported_handoff_schema", Detail: "batata does not declare support for handoff schema x"},
				{Dimension: "role_contract_version", Code: "unsupported_role_contract_version", Detail: "batata does not declare support for role contract v1"},
			},
		},
	}

	out := captureStdout(t, func() { printExcludedCandidates(excluded) })

	assert.Contains(t, out, "sdd-ask: excluded — role_mismatch: provider declares roles [sniper], role contract is \"archivist\"")
	assert.Contains(t, out, "batata: excluded — unsupported_handoff_schema: batata does not declare support for handoff schema x; unsupported_role_contract_version: batata does not declare support for role contract v1")
}

func TestPrintExcludedCandidatesNoopOnEmptyInput(t *testing.T) {
	// No t.Parallel(): captureStdout swaps the process-global os.Stdout, which
	// is unsafe to do concurrently with other tests.
	out := captureStdout(t, func() { printExcludedCandidates(nil) })
	assert.Empty(t, out)
}
