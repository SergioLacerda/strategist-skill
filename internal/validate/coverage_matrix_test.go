package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This matrix pins WHAT `strategist validate` catches today, across a complete
// runtime tree, so a refactor of the command or of this package cannot silently
// change its coverage. It mirrors the stages the command runs: active.yaml, every
// personas/*.yaml, every roles/*.yaml, and knowledge.index.yaml when present.
//
// A case with an empty wantErr documents a defect `validate` does NOT report
// even though it is invalid (name it ...KnownGap and say who reports it
// instead). Such gaps are pinned on purpose: closing one must be a deliberate
// change to this table, never a side effect. There are none left at the moment.

const (
	matrixActive = "mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-propose\n  execution: sniper\nprovider_resolution_policy: ask\n"
	matrixSlots  = "discovery: brainstorming\nrefinement: archivist\nexecution: sniper\n"
	matrixPerson = "id: %s\ntone_directive: precise\nphase_labels:\n  discovery: analysis\n  refinement: refinement\n  execution: execution\ndiagnostics:\n  pipeline_header: \"[Strategist] pipeline=starting\"\n  bootstrap_origin: \"[Strategist] profile_path={path}\"\n"
)

func writeMatrixFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}

// newMatrixRuntime builds a fully valid runtime tree.
func newMatrixRuntime(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeMatrixFile(t, root, "active.yaml", matrixActive)
	writeMatrixFile(t, root, "personas/epic.yaml", fmt.Sprintf(matrixPerson, "epic"))
	writeMatrixFile(t, root, "personas/pragmatic.yaml", fmt.Sprintf(matrixPerson, "pragmatic"))
	writeMatrixFile(t, root, "roles/default.yaml", matrixSlots)
	writeMatrixFile(t, root, "knowledge.index.yaml", "sources: []\n")
	return root
}

// validateTree runs the same stages, in the same order, as the validate command.
func validateTree(root string) []string {
	var errs []string
	if err := ActiveYAML(filepath.Join(root, "active.yaml")); err != nil {
		errs = append(errs, "active.yaml: "+err.Error())
	}
	personaErrs, _ := PersonasDir(filepath.Join(root, "personas"))
	errs = append(errs, personaErrs...)
	roleErrs, _ := RolesDir(filepath.Join(root, "roles"))
	errs = append(errs, roleErrs...)
	kiPath := filepath.Join(root, "knowledge.index.yaml")
	if _, err := os.Stat(kiPath); err == nil {
		if err := YAMLFile(kiPath); err != nil {
			errs = append(errs, "knowledge.index.yaml: "+err.Error())
		}
	}
	return errs
}

func TestValidateCoverageMatrix(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(t *testing.T, root string)
		wantErr string // substring of one reported error; "" means validate must report nothing
	}{
		{
			name:   "baseline valid runtime reports nothing",
			mutate: func(*testing.T, string) {},
		},
		{
			name: "non-active persona missing tone_directive",
			mutate: func(t *testing.T, root string) {
				writeMatrixFile(t, root, "personas/pragmatic.yaml",
					"id: pragmatic\nphase_labels:\n  discovery: a\n  refinement: b\n  execution: c\ndiagnostics:\n  pipeline_header: h\n  bootstrap_origin: o\n")
			},
			wantErr: "personas/pragmatic.yaml: persona config invalid: tone_directive is required",
		},
		{
			name: "role file with an unknown slot",
			mutate: func(t *testing.T, root string) {
				writeMatrixFile(t, root, "roles/bad.yaml", "role: x\nslot: nonsense\n")
			},
			wantErr: `roles/bad.yaml: role config invalid: slot "nonsense" is not one of`,
		},
		{
			name: "knowledge index that is not valid YAML",
			mutate: func(t *testing.T, root string) {
				writeMatrixFile(t, root, "knowledge.index.yaml", "a: [unclosed\n")
			},
			wantErr: "knowledge.index.yaml: invalid YAML",
		},
		{
			name: "mode outside pragmatic|epic",
			mutate: func(t *testing.T, root string) {
				writeMatrixFile(t, root, "active.yaml", "mode: foo\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n")
			},
			wantErr: `invalid mode "foo"`,
		},
		{
			name: "missing execution slot",
			mutate: func(t *testing.T, root string) {
				writeMatrixFile(t, root, "active.yaml", "mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-propose\n")
			},
			wantErr: "missing slot: execution",
		},
		{
			name: "bogus provider_resolution_policy",
			mutate: func(t *testing.T, root string) {
				writeMatrixFile(t, root, "active.yaml", "mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-propose\n  execution: sniper\nprovider_resolution_policy: bogus\n")
			},
			wantErr: `provider_resolution_policy "bogus"`,
		},
		{
			name: "invalid mode and missing slot are both reported",
			mutate: func(t *testing.T, root string) {
				writeMatrixFile(t, root, "active.yaml", "mode: foo\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n")
			},
			wantErr: "missing slot: execution",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := newMatrixRuntime(t)
			tc.mutate(t, root)

			assertReports(t, validateTree(root), tc.wantErr)
		})
	}
}

// assertReports checks that errs contains one entry with wantErr, or that errs
// is empty when wantErr is "" (a pinned KnownGap).
func assertReports(t *testing.T, errs []string, wantErr string) {
	t.Helper()
	if wantErr == "" {
		assert.Empty(t, errs)
		return
	}
	for _, e := range errs {
		if strings.Contains(e, wantErr) {
			return
		}
	}
	t.Errorf("want an error containing %q, got %v", wantErr, errs)
}
