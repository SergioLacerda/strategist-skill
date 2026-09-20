package runtimepayload

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const embeddedRuntimeDirPath = "skills/openspec-propose/runtime"

type buildInfo struct {
	Version      string `yaml:"version"`
	Bundle       string `yaml:"bundle"`
	BundleSHA256 string `yaml:"bundle_sha256"`
	TreeSHA256   string `yaml:"tree_sha256"`
	TreeBytes    int64  `yaml:"tree_bytes"`
}

func readBuildInfo(t *testing.T, raw []byte) buildInfo {
	t.Helper()
	var info buildInfo
	require.NoError(t, yaml.Unmarshal(raw, &info))
	return info
}

// The script that builds the runtime (Python) and the Go verifier must agree
// on the tree digest, or a freshly built runtime would never verify at install.
func TestEmbeddedOpenSpecRuntimeMatchesItsBuildInfo(t *testing.T) {
	src := embed.DefaultsFS()
	raw, err := fsReadFile(src, embeddedRuntimeDirPath+"/"+BuildInfoFile)
	require.NoError(t, err)
	info := readBuildInfo(t, raw)

	sum, size, err := TreeDigest(src, embeddedRuntimeDirPath)
	require.NoError(t, err)
	require.Equal(t, info.TreeSHA256, sum, "rebuild with scripts/build-openspec-runtime.sh and re-run prepare-embedded")
	require.Equal(t, info.TreeBytes, size)
	require.Equal(t, "1.13.0", info.Version, "must match runtime.version in strategist.yaml")
}

func TestSourceRuntimeIsIdenticalToEmbeddedMirror(t *testing.T) {
	root := filepath.Join("..", "..", "external-skills-source", "openspec-propose", "runtime")
	sourceSum, _, err := TreeDigest(os.DirFS(root), ".")
	require.NoError(t, err)
	mirrorSum, _, err := TreeDigest(embed.DefaultsFS(), embeddedRuntimeDirPath)
	require.NoError(t, err)
	require.Equal(t, sourceSum, mirrorSum, "run make embed-skills")
}

func fsReadFile(f interface{ Open(string) (fs.File, error) }, name string) ([]byte, error) {
	return fs.ReadFile(f.(fs.FS), name)
}
