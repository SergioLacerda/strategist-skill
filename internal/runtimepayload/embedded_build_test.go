package runtimepayload

import (
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/require"
)

func nodeFSFor(t *testing.T, target, version string) fstest.MapFS {
	t.Helper()
	archive := tarGz(t, entry{name: "bin/node", body: "#!/bin/sh\nexit 0\n", mode: 0o755}, entry{name: "LICENSE", body: "lic", mode: 0o644})
	info := "version: " + version + "\ntarget: " + target + "\nfile: node.tar.gz\nformat: tar.gz\nsha256: " + digest(archive) + "\nsize: " + strconv.Itoa(len(archive)) + "\n"
	return fstest.MapFS{
		"nodepayload/" + target + "/node.tar.gz":    {Data: archive},
		"nodepayload/" + target + "/node.info.yaml": {Data: []byte(info)},
	}
}

func TestBuildEmbeddedComposesRealOpenSpecTreeWithTargetNode(t *testing.T) {
	m, src, err := BuildEmbedded(embed.DefaultsFS(), nodeFSFor(t, "linux-amd64", "22.23.2"), "linux-amd64")
	require.NoError(t, err)
	require.NoError(t, m.Validate())
	require.Equal(t, "openspec-propose", m.Provider)

	dest := filepath.Join(t.TempDir(), "openspec-propose")
	ev, err := Materialize(src, m, "linux", "amd64", dest)
	require.NoError(t, err)
	require.Len(t, ev.Components, 2)

	node, script, err := m.LauncherPaths(dest, "linux")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dest, "node", "bin", "node"), node)
	require.FileExists(t, script)
	require.FileExists(t, filepath.Join(dest, "openspec", "schemas", "spec-driven", "schema.yaml"))
}

func TestBuildEmbeddedFailsClosedOnVersionOrTargetSkew(t *testing.T) {
	defaults := embed.DefaultsFS()

	_, _, err := BuildEmbedded(defaults, nodeFSFor(t, "linux-amd64", "20.0.0"), "linux-amd64")
	require.ErrorContains(t, err, "node version")

	_, _, err = BuildEmbedded(defaults, nodeFSFor(t, "linux-arm64", "22.23.2"), "linux-amd64")
	require.Error(t, err, "payload built for another target must be rejected")

	_, _, err = BuildEmbedded(defaults, fstest.MapFS{}, "linux-amd64")
	require.ErrorIs(t, err, ErrPayloadMissing)
}
