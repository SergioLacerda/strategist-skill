package telemetry

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRefinedPackagePublicationHistoryIsSeparateAndRoundTrips(t *testing.T) {
	root := t.TempDir()
	path := RefinedPackagePublicationHistoryPath(root)
	record := RefinedPackagePublicationRecord{
		MissionID: "m-publication", ProviderChangeID: "change-1",
		SourceDigest: "sha256:source", PackageDigest: "sha256:package",
		PublishedAt: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, AppendRefinedPackagePublication(path, record))
	got, err := ReadRefinedPackagePublications(path)
	require.NoError(t, err)
	require.Equal(t, []RefinedPackagePublicationRecord{record}, got)
	require.NoFileExists(t, filepath.Join(root, "memory", "sniper-materializations.jsonl"))
}

func TestRefinedPackagePublicationRequiresIdentity(t *testing.T) {
	err := AppendRefinedPackagePublication(filepath.Join(t.TempDir(), "history.jsonl"), RefinedPackagePublicationRecord{})
	require.ErrorContains(t, err, "source digest")
}
