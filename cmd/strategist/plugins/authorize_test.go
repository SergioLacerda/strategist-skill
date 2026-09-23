package plugins

import (
	"bytes"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/authorization"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAuthorize_RequiresTarget(t *testing.T) {
	t.Parallel()
	err := RunAuthorize(&bytes.Buffer{}, AuthorizeOptions{})
	require.EqualError(t, err, "plugins authorize: --target is required")
}

func TestAuthorizeCmd_BlockedReportKeepsSentinel(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, "plugins", "authorize", "--root", t.TempDir(), "--target", "docs/next.md", "--json")
	// errors.Is through RunE is what root's exitCodeFor relies on (exit 2).
	require.ErrorIs(t, err, authorization.ErrBlocked)
	assert.Contains(t, out, `"decision": "blocked"`)
	assert.Contains(t, out, `"reason_code": "runtime_unavailable"`)
}

func TestPrintAuthorization_Table(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	require.NoError(t, printAuthorization(&out, false, authorization.Report{
		Decision: "allowed", ReasonCode: "ok", Target: "docs/readme.md", Permission: "documentation",
		Dimensions: []authorization.Dimension{{Name: "runtime", Status: "ready", ReasonCode: "verified", EvidenceState: "static"}},
	}))
	assert.Contains(t, out.String(), "authorization=allowed reason=ok target=docs/readme.md permission=documentation")
	assert.Contains(t, out.String(), "runtime=ready reason=verified evidence=static")
}

func TestPrintAuthorization_JSON(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	require.NoError(t, printAuthorization(&out, true, authorization.Report{
		SchemaVersion: authorization.ReportSchemaVersion, Decision: "denied", ReasonCode: "blocked",
		Target: "src/main.go", Dimensions: []authorization.Dimension{},
	}))
	assert.Contains(t, out.String(), `"schema_version": "strategist-authorization-report/v1"`)
	assert.Contains(t, out.String(), `"target": "src/main.go"`)
}
