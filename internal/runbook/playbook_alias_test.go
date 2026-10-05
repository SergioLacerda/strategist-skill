package runbook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSidecarTreatsPlaybookAsOperationalRunbook(t *testing.T) {
	rb, err := ParseSidecar([]byte(`schema_version: runbook/v2
runbook_id: deploy
runbook_type: playbook
source_doc: docs/deploy.md
applies_when: [deploy]
objective: deploy safely
metadata:
  version: 1
  reviewed_at: 2026-10-05
`))
	require.NoError(t, err)
	assert.Equal(t, RunbookTypeOperational, rb.RunbookType)
}

func TestNormalizeRunbookTypeRejectsUnknownFamily(t *testing.T) {
	_, err := NormalizeRunbookType("checklist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runbook_type")
}
