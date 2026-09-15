package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errAllPathsLister wraps fakeUpgradeSource but fails AllPaths, to exercise
// PlanUpgrade's (and therefore extractRuntimeTree's) list-embedded-paths
// error branch.
type errAllPathsLister struct{ fakeUpgradeSource }

func (errAllPathsLister) AllPaths() ([]string, error) { return nil, errors.New("boom") }

func TestExtractRuntimeTree_ThreeWayMerge_FreshInstallWritesEverything(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("v1")})

	hashes, backupDir, err := svc.extractRuntimeTree(dir, false)
	require.NoError(t, err)
	assert.Empty(t, backupDir, "nothing existed yet to back up")
	assert.Equal(t, domain.SHA256Hex([]byte("v1")), hashes["a.yaml"])

	got, err := os.ReadFile(filepath.Join(dir, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "v1", string(got))
}

func TestExtractRuntimeTree_ThreeWayMerge_PlanUpgradeErrorPropagates(t *testing.T) {
	t.Parallel()

	src := fakeUpgradeSource{files: map[string][]byte{"a.yaml": []byte("v1")}}
	svc := Service{Extractor: src, Lister: errAllPathsLister{src}}

	_, _, err := svc.extractRuntimeTree(t.TempDir(), false)
	require.Error(t, err)
	require.ErrorContains(t, err, "extract runtime tree")
}

func TestExtractRuntimeTree_ThreeWayMerge_AutoUpgradeBacksUpAndRewrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc1 := upgradeTestService(map[string][]byte{"a.yaml": []byte("v1")})
	plan, err := svc1.PlanUpgrade(dir)
	require.NoError(t, err)
	_, err = svc1.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)

	// Embedded content changed; the on-disk file was never touched by the
	// user, so this run classifies as auto_upgrade, not customized.
	svc2 := upgradeTestService(map[string][]byte{"a.yaml": []byte("v2")})
	hashes, backupDir, err := svc2.extractRuntimeTree(dir, false)
	require.NoError(t, err)
	require.NotEmpty(t, backupDir, "an auto-upgraded file must be snapshotted first")
	assert.Equal(t, domain.SHA256Hex([]byte("v2")), hashes["a.yaml"])

	got, err := os.ReadFile(filepath.Join(dir, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "v2", string(got))

	backedUp, err := os.ReadFile(filepath.Join(backupDir, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "v1", string(backedUp))
}

func TestBuildInstallManifest(t *testing.T) {
	t.Parallel()

	t.Run("full hashes present uses the full-tree manifest shape", func(t *testing.T) {
		t.Parallel()
		manifest := buildInstallManifest("pkg", runtimeDefaultPlan{}, map[string]string{"a.yaml": "hash-a"})
		_, ok := manifest.FileByPath("a.yaml")
		assert.True(t, ok)
	})

	t.Run("nil full hashes falls back to the narrower normative-only manifest", func(t *testing.T) {
		t.Parallel()
		manifest := buildInstallManifest("pkg", runtimeDefaultPlan{}, nil)
		assert.NotNil(t, manifest)
	})
}
