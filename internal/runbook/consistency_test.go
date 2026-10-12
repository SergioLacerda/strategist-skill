package runbook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func consistencyRoot(t *testing.T, docs ...string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "runbooks")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	for _, doc := range docs {
		require.NoError(t, os.WriteFile(filepath.Join(dir, doc), []byte("# doc\n"), 0o600))
	}
	return root
}

func codes(findings []Finding) map[string]string {
	out := make(map[string]string, len(findings))
	for _, f := range findings {
		out[f.Code] = f.Severity
	}
	return out
}

func TestCheckSidecarConsistencyAcceptsAConsistentSidecar(t *testing.T) {
	root := consistencyRoot(t, "alpha.md")
	rb := Runbook{RunbookID: "alpha", SourceDoc: "docs/runbooks/alpha.md"}
	assert.Empty(t, CheckSidecarConsistency("docs/runbooks/alpha.runbook.yaml", rb, root))
}

func TestCheckSidecarConsistencySkipsAMissingSourceDoc(t *testing.T) {
	root := consistencyRoot(t)
	rb := Runbook{RunbookID: "alpha", SourceDoc: "docs/runbooks/alpha.md"}
	got := codes(CheckSidecarConsistency("docs/runbooks/alpha.runbook.yaml", rb, root))
	assert.Equal(t, SeveritySkip, got[CodeSourceDocMissing])
}

func TestCheckSidecarConsistencyWarnsWhenSourceDocIsNotMarkdown(t *testing.T) {
	root := consistencyRoot(t, "alpha.runbook.yaml")
	rb := Runbook{RunbookID: "alpha", SourceDoc: "docs/runbooks/alpha.runbook.yaml"}
	got := codes(CheckSidecarConsistency("docs/runbooks/alpha.runbook.yaml", rb, root))
	assert.Equal(t, map[string]string{CodeSourceDocNotMarkdown: SeverityWarn}, got)
}

func TestCheckSidecarConsistencyWarnsOnStemMismatch(t *testing.T) {
	root := consistencyRoot(t, "alpha.md")
	rb := Runbook{RunbookID: "beta", SourceDoc: "docs/runbooks/alpha.md"}
	got := codes(CheckSidecarConsistency("docs/runbooks/alpha.runbook.yaml", rb, root))
	assert.Equal(t, map[string]string{CodeIDStemMismatch: SeverityWarn}, got)
}

func TestCheckSidecarConsistencyTreatsAnEscapingSourceDocAsMissing(t *testing.T) {
	root := consistencyRoot(t, "alpha.md")
	for _, sd := range []string{"../outside.md", "/etc/passwd.md"} {
		rb := Runbook{RunbookID: "alpha", SourceDoc: sd}
		got := codes(CheckSidecarConsistency("docs/runbooks/alpha.runbook.yaml", rb, root))
		assert.Equal(t, SeveritySkip, got[CodeSourceDocMissing], sd)
	}
}
