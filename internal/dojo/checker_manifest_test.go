package dojo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckManifests_Pass(t *testing.T) {
	strategistDir := t.TempDir()
	providerDir := filepath.Join(strategistDir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "skill.yaml"),
		[]byte("canonical_role: ranger\nprovider_class: rankeado\n"), 0o644))

	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{
				Slot:             "discovery",
				ExpectedProvider: "brainstorming",
				ManifestExists:   true,
				FieldsPresent:    []string{"canonical_role", "provider_class"},
			},
		},
	}
	items := dojo.CheckManifests(criteria, strategistDir)
	for _, it := range items {
		assert.True(t, it.Passed, "expected pass: %s — %s", it.Label, it.Detail)
	}
}

func TestCheckManifests_ManifestMissing(t *testing.T) {
	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{Slot: "discovery", ExpectedProvider: "brainstorming", ManifestExists: true},
		},
	}
	items := dojo.CheckManifests(criteria, t.TempDir())
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}

func TestCheckManifests_ManifestExpectedAbsent_IsAbsent(t *testing.T) {
	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{Slot: "execution", ExpectedProvider: "sdd-ask", ManifestExists: false},
		},
	}
	items := dojo.CheckManifests(criteria, t.TempDir())
	require.Len(t, items, 1)
	assert.True(t, items[0].Passed)
}

func TestCheckManifests_FieldMissing(t *testing.T) {
	strategistDir := t.TempDir()
	providerDir := filepath.Join(strategistDir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "skill.yaml"),
		[]byte("id: brainstorming\n"), 0o644))

	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{
				Slot:             "discovery",
				ExpectedProvider: "brainstorming",
				ManifestExists:   true,
				FieldsPresent:    []string{"canonical_role"},
			},
		},
	}
	items := dojo.CheckManifests(criteria, strategistDir)
	failed := false
	for _, it := range items {
		if !it.Passed {
			failed = true
		}
	}
	assert.True(t, failed)
}

func TestCheckManifests_NestedFieldPresent(t *testing.T) {
	strategistDir := t.TempDir()
	providerDir := filepath.Join(strategistDir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "skill.yaml"),
		[]byte("specialization_taxonomy:\n  canonical_role: ranger\n"), 0o644))

	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{
				Slot:             "discovery",
				ExpectedProvider: "brainstorming",
				ManifestExists:   true,
				FieldsPresent:    []string{"specialization_taxonomy.canonical_role"},
			},
		},
	}
	items := dojo.CheckManifests(criteria, strategistDir)
	for _, it := range items {
		assert.True(t, it.Passed, "expected pass: %s — %s", it.Label, it.Detail)
	}
}

func TestCheckManifests_FieldInsideListIsFound(t *testing.T) {
	// Exercises manifestSliceHasKeyAnywhere: a plain (non-dotted) field lookup
	// must also search inside YAML sequences, not just nested maps.
	strategistDir := t.TempDir()
	providerDir := filepath.Join(strategistDir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "skill.yaml"),
		[]byte("items:\n  - name: unrelated\n  - canonical_role: ranger\n"), 0o644))

	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{
				Slot:             "discovery",
				ExpectedProvider: "brainstorming",
				ManifestExists:   true,
				FieldsPresent:    []string{"canonical_role"},
			},
		},
	}
	items := dojo.CheckManifests(criteria, strategistDir)
	require.NotEmpty(t, items)
	assert.True(t, items[len(items)-1].Passed, "expected canonical_role nested inside a list to be found")
}

func TestCheckManifests_FieldAbsentFromList(t *testing.T) {
	// Exercises manifestSliceHasKeyAnywhere's not-found path: a list whose
	// entries never contain the requested field.
	strategistDir := t.TempDir()
	providerDir := filepath.Join(strategistDir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "skill.yaml"),
		[]byte("items:\n  - name: unrelated\n  - other: value\n"), 0o644))

	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{
				Slot:             "discovery",
				ExpectedProvider: "brainstorming",
				ManifestExists:   true,
				FieldsPresent:    []string{"canonical_role"},
			},
		},
	}
	items := dojo.CheckManifests(criteria, strategistDir)
	require.NotEmpty(t, items)
	assert.False(t, items[len(items)-1].Passed, "expected canonical_role absent from every list entry to fail")
}

func TestCheckManifests_NestedFieldMissing(t *testing.T) {
	strategistDir := t.TempDir()
	providerDir := filepath.Join(strategistDir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "skill.yaml"),
		[]byte("specialization_taxonomy:\n  other_field: x\n"), 0o644))

	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{
				Slot:             "discovery",
				ExpectedProvider: "brainstorming",
				ManifestExists:   true,
				FieldsPresent:    []string{"specialization_taxonomy.canonical_role"},
			},
		},
	}
	items := dojo.CheckManifests(criteria, strategistDir)
	require.NotEmpty(t, items)
	assert.False(t, items[len(items)-1].Passed)
}

func TestCheckManifests_FieldSubstringNoLongerFalsePositive(t *testing.T) {
	// canonical_role_extended must not satisfy a field check for "canonical_role"
	// now that manifest checks parse structured YAML instead of substring search.
	strategistDir := t.TempDir()
	providerDir := filepath.Join(strategistDir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "skill.yaml"),
		[]byte("canonical_role_extended: ranger\n"), 0o644))

	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{
				Slot:             "discovery",
				ExpectedProvider: "brainstorming",
				ManifestExists:   true,
				FieldsPresent:    []string{"canonical_role"},
			},
		},
	}
	items := dojo.CheckManifests(criteria, strategistDir)
	require.NotEmpty(t, items)
	assert.False(t, items[len(items)-1].Passed)
}

func TestCheckManifests_Empty(t *testing.T) {
	criteria := domain.DojoCriteria{}
	items := dojo.CheckManifests(criteria, t.TempDir())
	assert.Empty(t, items)
}
