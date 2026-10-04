package customws_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/check"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/SergioLacerda/strategist-skill/internal/rolevalidation"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type noCompile struct{}

func (noCompile) CompileAll(_, _ string) error { return nil }

const hostWeaponID = "team-brainstorming"

// hostSkill writes a host skill under the workspace, with or without the
// Strategist sidecar the wizard now requires.
func hostSkill(t *testing.T, workspace string, withSidecar bool) string {
	t.Helper()
	dir := filepath.Join(workspace, ".agents", "skills", hostWeaponID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+hostWeaponID+"\nmetadata:\n  version: \"2.0.0\"\n---\nbody\n"), 0o644))
	if withSidecar {
		_, err := install.ScaffoldSidecar(install.SidecarScaffoldOptions{PackageDir: dir, Roles: []string{"ranger"}, Slots: []string{"discovery"}})
		require.NoError(t, err)
	}
	return dir
}

func wizardInstall(t *testing.T, workspace string) error {
	t.Helper()
	input := "en\nen\nen\nen\nepic\n.analysis\n" + hostWeaponID + "\nopenspec-propose\nsniper\n\n"
	svc := install.Service{Extractor: embed.Extractor{}, Compiler: noCompile{}, WizardPrompter: install.NewTextPrompter(strings.NewReader(input)), ShimHomeDir: t.TempDir()}
	return svc.Install(context.Background(), domain.InstallConfig{Target: workspace, Wizard: true})
}

func readLock(t *testing.T, root string) domain.PluginLockFile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, err)
	var lock domain.PluginLockFile
	require.NoError(t, yaml.Unmarshal(raw, &lock))
	return lock
}

// runCheck runs `strategist check --json` against root and returns its report
// and the command error: a blocked check prints its report to stdout and then
// fails, so stdout is captured around the call.
func runCheck(t *testing.T, root string) (map[string]any, error) {
	t.Helper()
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = writer
	cmd := &cobra.Command{Use: "strategist", SilenceUsage: true, SilenceErrors: true}
	check.Register(cmd)
	cmd.SetOut(writer)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"check", "--root", root, "--json"})
	runErr := cmd.Execute()
	os.Stdout = original
	require.NoError(t, writer.Close())
	var out bytes.Buffer
	_, err = out.ReadFrom(reader)
	require.NoError(t, err)
	var report map[string]any
	if start := bytes.IndexByte(out.Bytes(), '{'); start >= 0 {
		require.NoError(t, json.NewDecoder(bytes.NewReader(out.Bytes()[start:])).Decode(&report), out.String())
	}
	return report, runErr
}

func TestWizardTypedCustomWeaponIsInstalledCheckedAndPlannableFromTheInstalledPackage(t *testing.T) {
	workspace := t.TempDir()
	testutil.SetHome(t, t.TempDir())
	t.Chdir(workspace)
	host := hostSkill(t, workspace, true)

	require.NoError(t, wizardInstall(t, workspace))

	root := filepath.Join(workspace, ".strategist")
	instance := hostWeaponID + "@2.0.0"
	assert.FileExists(t, filepath.Join(root, "providers", instance, "package.yaml"))
	assert.FileExists(t, filepath.Join(root, "providers", instance, "adapter.yaml"))
	active, err := os.ReadFile(filepath.Join(root, "active.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(active), "discovery: "+instance, "active.yaml names the versioned instance id")

	lock := readLock(t, root)
	binding, err := domain.SingleLockBindingForSlot(lock, "discovery")
	require.NoError(t, err)
	require.NoError(t, domain.ValidateCustomBinding(lock, binding, "ranger", "discovery"))
	assert.Equal(t, instance, binding.InstalledInstanceID)
	assert.Equal(t, "host.prompt", binding.Entrypoint, "the binding never records the transient host path")
	assert.NotContains(t, binding.Entrypoint, host)

	plan, err := rolevalidation.BuildRoleInvocationPlan(root, "discovery")
	require.NoError(t, err)
	assert.Equal(t, instance, plan.WeaponID)
	assert.Equal(t, binding.WeaponDigest, plan.WeaponDigest)
	assert.Equal(t, binding.BindingDigest, plan.BindingDigest)
	assert.Equal(t, domain.RankedRuntimeHost, plan.Runtime.Kind)

	// The refinement slot is an unrelated catalog runtime that this hermetic
	// workspace does not provision, so the report as a whole may be blocked;
	// the Custom discovery binding is what this test is about.
	before, _ := runCheck(t, root) //nolint:errcheck // the report is asserted per slot.
	assert.Equal(t, "ready", slotStatus(t, before, "discovery"))
	require.NoError(t, os.RemoveAll(host), "the original host directory disappears after install")
	after, _ := runCheck(t, root) //nolint:errcheck // the report is asserted per slot.
	assert.Equal(t, "ready", slotStatus(t, after, "discovery"), "check evaluates only the installed representation")
	_, err = rolevalidation.BuildRoleInvocationPlan(root, "discovery")
	require.NoError(t, err)
}

func TestWizardTypedCustomWeaponWithoutASidecarStopsBeforeAnyMutation(t *testing.T) {
	workspace := t.TempDir()
	testutil.SetHome(t, t.TempDir())
	t.Chdir(workspace)
	hostSkill(t, workspace, false)

	err := wizardInstall(t, workspace)

	require.ErrorContains(t, err, "sidecar")
	require.ErrorContains(t, err, "scaffold-sidecar")
	assert.NoDirExists(t, filepath.Join(workspace, ".strategist", "providers", hostWeaponID+"@2.0.0"))
	assert.NoFileExists(t, filepath.Join(workspace, ".strategist", "plugins.lock"))
}

func TestWizardTypedCustomWeaponDeclaringAnotherRoleIsRejected(t *testing.T) {
	workspace := t.TempDir()
	testutil.SetHome(t, t.TempDir())
	t.Chdir(workspace)
	dir := hostSkill(t, workspace, false)
	_, err := install.ScaffoldSidecar(install.SidecarScaffoldOptions{PackageDir: dir, Roles: []string{"archivist"}, Slots: []string{"refinement"}})
	require.NoError(t, err)

	err = wizardInstall(t, workspace)

	require.ErrorContains(t, err, "declares Roles")
	assert.NoDirExists(t, filepath.Join(workspace, ".strategist", "providers", hostWeaponID+"@2.0.0"))
}

func slotStatus(t *testing.T, report map[string]any, slot string) string {
	t.Helper()
	bindings, _ := report["bindings"].([]any) //nolint:errcheck // a missing list fails the lookup below.
	for _, raw := range bindings {
		binding, _ := raw.(map[string]any) //nolint:errcheck // a malformed entry is skipped.
		if binding["slot"] == slot {
			status, _ := binding["status"].(string) //nolint:errcheck // an absent status reads as empty.
			return status
		}
	}
	require.FailNow(t, "no binding for slot "+slot)
	return ""
}

// providerAddPackage is a Custom package equivalent to the wizard's host Weapon
// (same id, version, Role and slot), in the form `provider add` consumes.
func providerAddPackage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.yaml"), []byte("schema_version: strategist-plugin-package/v1\nid: "+hostWeaponID+"\nversion: 2.0.0\ndigest: sha256:"+strings.Repeat("a", 64)+"\nartifact_uri: local://team\nartifact_size: 64\nlicense: Apache-2.0\ncreated_at: 2026-09-20T00:00:00Z\nmanifest_schema: orka/v1\nupstream_version: 2.0.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "adapter.yaml"), []byte("schema_version: strategist-plugin-adapter/v1\nid: "+hostWeaponID+"\nadapter_revision: 2.0.0\nplugin_api_range: \">=1 <2\"\nsupported_slots: [discovery]\nsupported_roles: [ranger]\nentrypoints: [host.prompt]\npackage_constraint: "+hostWeaponID+"@2\nrequested_permissions: []\nrisk_score: write_analysis\n"), 0o644))
	return dir
}

// bindingShape is a binding's acquisition-independent structure: every
// identity field is present and the same, while digests and the connector may
// legitimately differ by how the package was acquired.
type bindingShape struct {
	Slot, Instance, Role, Version, Origin, Runtime, Entrypoint, Mode, Status string
	HasDigests, HasConnector                                                 bool
}

func shapeOf(b domain.SlotBinding) bindingShape {
	return bindingShape{
		Slot: b.Slot, Instance: b.InstalledInstanceID, Role: b.Role, Version: b.WeaponVersion, Origin: b.Origin,
		Runtime: b.RuntimeKind, Entrypoint: b.Entrypoint, Mode: b.Mode, Status: b.Status,
		HasDigests:   b.WeaponDigest != "" && b.SourceDigest != "" && b.BindingDigest != "",
		HasConnector: b.ConnectorID != "",
	}
}

func nodeShape(lock domain.PluginLockFile) []string {
	var shape []string
	for _, node := range lock.Lock.Nodes {
		if strings.Contains(node.ID, hostWeaponID) {
			shape = append(shape, node.Kind+"|"+node.ID)
		}
	}
	sort.Strings(shape)
	return shape
}

func TestProviderAddAndWizardPublishTheSameCustomShapes(t *testing.T) {
	// Wizard acquisition.
	wizardWorkspace := t.TempDir()
	testutil.SetHome(t, t.TempDir())
	t.Chdir(wizardWorkspace)
	hostSkill(t, wizardWorkspace, true)
	require.NoError(t, wizardInstall(t, wizardWorkspace))
	wizardRoot := filepath.Join(wizardWorkspace, ".strategist")

	// provider add acquisition, into a runtime extracted from the same defaults.
	addRoot := t.TempDir()
	require.NoError(t, embed.Extractor{}.Extract(addRoot, true))
	require.NoError(t, os.WriteFile(filepath.Join(addRoot, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\nknowledge_index_path: knowledge.index.yaml\nslots:\n  discovery: "+hostWeaponID+"@2.0.0\n  refinement: archivist\n  execution: sniper\n"), 0o644))
	_, err := provider.Add(addRoot, providerAddPackage(t), "discovery")
	require.NoError(t, err)

	wizardLock, addLock := readLock(t, wizardRoot), readLock(t, addRoot)
	wizardBinding, err := domain.SingleLockBindingForSlot(wizardLock, "discovery")
	require.NoError(t, err)
	addBinding, err := domain.SingleLockBindingForSlot(addLock, "discovery")
	require.NoError(t, err)
	require.NoError(t, domain.ValidateCustomBinding(wizardLock, wizardBinding, "ranger", "discovery"))
	require.NoError(t, domain.ValidateCustomBinding(addLock, addBinding, "ranger", "discovery"))

	assert.Equal(t, shapeOf(addBinding), shapeOf(wizardBinding), "both acquisition paths publish the same slot-binding shape")
	assert.Equal(t, nodeShape(addLock), nodeShape(wizardLock), "both publish the same package, adapter and role-binding nodes")

	instance := hostWeaponID + "@2.0.0"
	for _, root := range []string{wizardRoot, addRoot} {
		entries, err := os.ReadDir(filepath.Join(root, "providers", instance))
		require.NoError(t, err)
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		assert.ElementsMatch(t, []string{"package.yaml", "adapter.yaml"}, names)
		rawPackage, err := os.ReadFile(filepath.Join(root, "providers", instance, "package.yaml"))
		require.NoError(t, err)
		var pkg domain.PluginPackage
		require.NoError(t, provider.DecodeStrictPluginYAML(rawPackage, &pkg))
		require.NoError(t, pkg.Validate())
		rawAdapter, err := os.ReadFile(filepath.Join(root, "providers", instance, "adapter.yaml"))
		require.NoError(t, err)
		var adapter domain.AdapterContract
		require.NoError(t, provider.DecodeStrictPluginYAML(rawAdapter, &adapter))
		require.NoError(t, adapter.Validate())
		assert.Equal(t, []string{"host.prompt"}, adapter.Entrypoints)
		assert.Equal(t, []string{"ranger"}, adapter.SupportedRoles)
	}

	// Both are consumed by the same canonical planning boundary.
	for _, root := range []string{wizardRoot, addRoot} {
		_, err := rolevalidation.BuildRoleInvocationPlan(root, "discovery")
		require.NoError(t, err, root)
	}
}
