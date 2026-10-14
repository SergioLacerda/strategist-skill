package metrics

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A claim written by Windows PowerShell 5.1 carries a UTF-8 byte order mark;
// it must decode exactly like the same claim without the mark.
func TestReadClaimInputFromStdinAcceptsByteOrderMark(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("\xEF\xBB\xBFclaim:\n  id: c1\n  statement: s\n"))

	input, err := readClaimInput(cmd, Dependencies{}, "", claimFromStdin)

	require.NoError(t, err)
	assert.Equal(t, "c1", input.Claim.ID)
}

func TestReadDeclaredSummaryAcceptsByteOrderMark(t *testing.T) {
	doc := "confidence_summary:\n  policy_version: v1\n  claims: []\n"
	_, withoutMark := readDeclaredSummary(strings.NewReader(doc))
	_, withMark := readDeclaredSummary(strings.NewReader("\xEF\xBB\xBF" + doc))

	assert.Equal(t, withoutMark == nil, withMark == nil, "a byte order mark must not change the result: %v vs %v", withoutMark, withMark)
}

func TestReadMissionQualityDocumentAcceptsByteOrderMark(t *testing.T) {
	_, withoutMark := readMissionQualityDocument(strings.NewReader("decisions: []\nevidence: []\n"))
	_, withMark := readMissionQualityDocument(strings.NewReader("\xEF\xBB\xBFdecisions: []\nevidence: []\n"))

	assert.Equal(t, withoutMark == nil, withMark == nil, "a byte order mark must not change the result: %v vs %v", withoutMark, withMark)
}
