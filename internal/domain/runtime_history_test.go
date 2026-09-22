package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestWithHistoryFromRecordsEarlierHashesMostRecentFirst(t *testing.T) {
	prev := domain.InstallManifest{Files: []domain.InstallManifestFile{{Path: "a", SHA256: "h2", History: []string{"h1"}}}}
	next := domain.InstallManifest{Files: []domain.InstallManifestFile{{Path: "a", SHA256: "h3"}, {Path: "new", SHA256: "n1"}}}

	got := next.WithHistoryFrom(prev, true)
	assert.Equal(t, []string{"h2", "h1"}, got.Files[0].History)
	assert.Empty(t, got.Files[1].History, "a new path has no history")

	rolledBack := domain.InstallManifest{Files: []domain.InstallManifestFile{{Path: "a", SHA256: "h1"}}}.WithHistoryFrom(got, true)
	assert.Equal(t, []string{"h3", "h2"}, rolledBack.Files[0].History, "the current hash never appears in its own history")
	assert.Equal(t, next, next.WithHistoryFrom(domain.InstallManifest{}, false), "no previous manifest, no history")
}

func TestDecideRuntimeDefaultUpdateDetectsADowngrade(t *testing.T) {
	base := domain.RuntimeDefaultDecisionInput{Exists: true, CurrentHash: "h2", ManifestHash: "h2", ManifestHistory: []string{"h1"}, HasManifest: true}
	older, newer := base, base
	older.EmbeddedHash, newer.EmbeddedHash = "h1", "h3"
	assert.Equal(t, domain.RuntimeDecisionDowngrade, domain.DecideRuntimeDefaultUpdate(older))
	assert.Equal(t, domain.RuntimeDecisionAutoUpgrade, domain.DecideRuntimeDefaultUpdate(newer))

	forced := older
	forced.Force = true
	assert.Equal(t, domain.RuntimeDecisionDowngrade, domain.DecideRuntimeDefaultUpdate(forced), "--force is not a downgrade permission")
	allowed := older
	allowed.AllowDowngrade = true
	assert.Equal(t, domain.RuntimeDecisionAutoUpgrade, domain.DecideRuntimeDefaultUpdate(allowed))
	assert.Contains(t, domain.FormatRuntimeStaleDiagnostic("x", domain.RuntimeDecisionDowngrade), "runtime_newer_than_binary")
}
