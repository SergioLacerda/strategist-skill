package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestDecideRuntimeDefaultUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   domain.RuntimeDefaultDecisionInput
		want domain.RuntimeDefaultDecision
	}{
		{
			name: "force overwrites",
			in:   domain.RuntimeDefaultDecisionInput{Force: true},
			want: domain.RuntimeDecisionForceOverwrite,
		},
		{
			name: "missing writes",
			in:   domain.RuntimeDefaultDecisionInput{Exists: false},
			want: domain.RuntimeDecisionWriteMissing,
		},
		{
			name: "current embedded keeps",
			in: domain.RuntimeDefaultDecisionInput{
				Exists:       true,
				CurrentHash:  "new",
				EmbeddedHash: "new",
			},
			want: domain.RuntimeDecisionKeepCurrent,
		},
		{
			name: "previous default upgrades",
			in: domain.RuntimeDefaultDecisionInput{
				Exists:       true,
				CurrentHash:  "old",
				EmbeddedHash: "new",
				ManifestHash: "old",
				HasManifest:  true,
			},
			want: domain.RuntimeDecisionAutoUpgrade,
		},
		{
			name: "local edit conflicts",
			in: domain.RuntimeDefaultDecisionInput{
				Exists:       true,
				CurrentHash:  "local",
				EmbeddedHash: "new",
				ManifestHash: "old",
				HasManifest:  true,
			},
			want: domain.RuntimeDecisionConflict,
		},
		{
			name: "missing manifest is unknown",
			in: domain.RuntimeDefaultDecisionInput{
				Exists:       true,
				CurrentHash:  "old",
				EmbeddedHash: "new",
			},
			want: domain.RuntimeDecisionUnknownManifest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, domain.DecideRuntimeDefaultUpdate(tt.in))
		})
	}
}

func TestNewInstallManifestRecordsNormativeDefaults(t *testing.T) {
	t.Parallel()

	hashes := map[string]string{}
	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		hashes[file.Path] = domain.SHA256Hex([]byte(file.Path))
	}

	manifest := domain.NewInstallManifest("test", hashes)
	assert.Equal(t, "strategist.install-manifest.v1", manifest.Schema)
	assert.Equal(t, "test", manifest.PackageID)
	assert.Len(t, manifest.Files, len(domain.NormativeRuntimeDefaultFiles()))

	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		got, ok := manifest.FileByPath(file.Path)
		assert.True(t, ok)
		assert.Equal(t, file.Owner, got.Owner)
		assert.Equal(t, hashes[file.Path], got.SHA256)
	}
}

func TestFileByPath_NotFound(t *testing.T) {
	t.Parallel()
	manifest := domain.NewInstallManifest("test", nil)
	_, ok := manifest.FileByPath("does/not/exist.yaml")
	assert.False(t, ok)
}

func TestNormativeRuntimeDefaultPaths_MatchesFiles(t *testing.T) {
	t.Parallel()
	files := domain.NormativeRuntimeDefaultFiles()
	paths := domain.NormativeRuntimeDefaultPaths()
	assert.Len(t, paths, len(files))
	for i, file := range files {
		assert.Equal(t, file.Path, paths[i])
	}
}

func TestFormatRuntimeStaleDiagnostic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		decision domain.RuntimeDefaultDecision
		want     string
	}{
		{"auto upgrade", domain.RuntimeDecisionAutoUpgrade, "runtime_stale_auto_repairable"},
		{"conflict", domain.RuntimeDecisionConflict, "runtime_stale_conflict"},
		{"unknown manifest", domain.RuntimeDecisionUnknownManifest, "runtime_stale_unknown_manifest"},
		{"write missing", domain.RuntimeDecisionWriteMissing, "runtime_stale:"},
		{"keep current", domain.RuntimeDecisionKeepCurrent, "runtime_stale:"},
		{"force overwrite", domain.RuntimeDecisionForceOverwrite, "runtime_stale:"},
		{"unrecognized decision", domain.RuntimeDefaultDecision("bogus"), "runtime_stale:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := domain.FormatRuntimeStaleDiagnostic("SKILL.md", tt.decision)
			assert.Contains(t, got, tt.want)
			assert.Contains(t, got, "SKILL.md")
		})
	}
}
