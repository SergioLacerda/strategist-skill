package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rangerWithMandatoryRunbook = `role: ranger
slot: discovery
canonical:
  - find_unexpected_items
  - select_runbook:
      must_invoke: true
      invoke: "strategist runbook select --mission {mission_id} --format json --signal <signal>"
`

func envOf(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestSupportBindingAdvisoryReportsMissingJewelerExecutable(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "roles/ranger.yaml", rangerWithMandatoryRunbook)

	got := supportBindingAdvisories(root, envOf(nil))

	require.Len(t, got, 1)
	assert.Contains(t, got[0], "reason=support_binding_unready")
	assert.Contains(t, got[0], "role=ranger")
	assert.Contains(t, got[0], "feat=select_runbook")
	assert.Contains(t, got[0], "detail=executable_not_configured")
	assert.Contains(t, got[0], "STRATEGIST_TREASURE_CHEST_EXECUTABLE")
}

func TestSupportBindingAdvisoryReportsExecutableThatDoesNotExist(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "roles/ranger.yaml", rangerWithMandatoryRunbook)

	got := supportBindingAdvisories(root, envOf(map[string]string{
		"STRATEGIST_TREASURE_CHEST_EXECUTABLE": filepath.Join(root, "missing-provider"),
	}))

	require.Len(t, got, 1)
	assert.Contains(t, got[0], "detail=executable_not_found")
}

func TestSupportBindingAdvisoryReportsInvalidCatalogBinding(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "roles/ranger.yaml", rangerWithMandatoryRunbook)
	exe := filepath.Join(root, "provider")
	require.NoError(t, os.WriteFile(exe, []byte("x"), 0o700))

	got := supportBindingAdvisories(root, envOf(map[string]string{"STRATEGIST_TREASURE_CHEST_EXECUTABLE": exe}))

	require.Len(t, got, 1)
	assert.Contains(t, got[0], "detail=catalog_unreadable")
}

func TestSupportBindingAdvisoryAcceptsHealthyBindingWithoutInvokingIt(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "roles/ranger.yaml", rangerWithMandatoryRunbook)
	shipped, err := embedpkg.Extractor{}.ReadFile("plugins/catalog.yaml") // TestMain replaces embeddedCatalogReader with a fixture
	require.NoError(t, err)
	writeRuntimeFile(t, root, "plugins/catalog.yaml", string(shipped))
	// A file that is not a runnable program: the check must not execute it.
	exe := filepath.Join(root, "provider-not-runnable")
	require.NoError(t, os.WriteFile(exe, []byte("not a program"), 0o600))

	got := supportBindingAdvisories(root, envOf(map[string]string{"STRATEGIST_TREASURE_CHEST_EXECUTABLE": exe}))

	assert.Empty(t, got)
}

func TestPreflightAdvisoriesIncludeSupportBindingReadiness(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "roles/ranger.yaml", rangerWithMandatoryRunbook)
	previous := supportBindingReadiness
	supportBindingReadiness = func(root string) []string { return supportBindingAdvisories(root, envOf(nil)) }
	t.Cleanup(func() { supportBindingReadiness = previous })

	got := preflightAdvisories(root)

	assert.Contains(t, strings.Join(got, "\n"), "reason=support_binding_unready")
}

func TestSupportBindingAdvisoryIgnoresRolesWithoutAMandatoryRunbookFeat(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "roles/sniper.yaml", "role: sniper\nslot: execution\ncanonical:\n  - materialize_one_target_per_loop\n")
	writeRuntimeFile(t, root, "roles/archivist.yaml", "role: archivist\nslot: refinement\ncanonical:\n  - select_runbook:\n      must_invoke: false\n")

	assert.Empty(t, supportBindingAdvisories(root, envOf(nil)))
}
