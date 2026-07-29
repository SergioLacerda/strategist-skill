package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMissionTasks_InvalidYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte("side_quests_approved:\n  : not: valID:\n"), 0o644))

	_, err := ParseMissionTasks("mission-x", path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse side_quests_approved")
}

func TestParseMissionTasks_FencedSideQuestsYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte(`
## Archivist-To-Sniper Handoff Fields

side_quests_approved:

`+"```yaml"+`
- id: SQ-001
  description: Keep fenced side quests parseable.
  strategy: execute_later
  estimated_impact: low
  dependencies: []
  status: sq_pending
`+"```"+`

acceptance_checks:

`+"```yaml"+`
- "unrelated fenced YAML remains outside side_quests_approved"
`+"```"+`
`), 0o644))

	mission, err := ParseMissionTasks("mission-x", path)
	require.NoError(t, err)
	require.Len(t, mission.SQs, 1)
	assert.Equal(t, "SQ-001", mission.SQs[0].ID)
	assert.Equal(t, "sq_pending", mission.SQs[0].Status)
}

func TestParseMissionTasks_FencedSideQuestsYAMLWithBacktickScalars(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte("side_quests_approved:\n\n```yml\n- id: `SQ-002`\n  description: Backtick scalars still parse.\n  dependencies: [`SQ-001`]\n  status: `sq_pending`\n```\n"), 0o644))

	mission, err := ParseMissionTasks("mission-x", path)
	require.NoError(t, err)
	require.Len(t, mission.SQs, 1)
	assert.Equal(t, "SQ-002", mission.SQs[0].ID)
	assert.Equal(t, []string{"SQ-001"}, mission.SQs[0].Dependencies)
	assert.Equal(t, "sq_pending", mission.SQs[0].Status)
}

func TestParseMissionTasks_LegacySideQuestFieldBullets(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte("side_quests_approved:\n\n- id: `SQ-003`\n  - description: Legacy field bullets still parse.\n  - strategy: `execute_later`\n  - estimated_impact: `medium`\n  - dependencies: none\n  - status: `sq_backlog`\n"), 0o644))

	mission, err := ParseMissionTasks("mission-x", path)
	require.NoError(t, err)
	require.Len(t, mission.SQs, 1)
	assert.Equal(t, "SQ-003", mission.SQs[0].ID)
	assert.Equal(t, "Legacy field bullets still parse.", mission.SQs[0].Description)
	assert.Equal(t, "execute_later", mission.SQs[0].Strategy)
	assert.Empty(t, mission.SQs[0].Dependencies)
	assert.Equal(t, "sq_backlog", mission.SQs[0].Status)
}

func TestParseMissionTasks_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := ParseMissionTasks("mission-x", filepath.Join(t.TempDir(), "nope.md"))
	require.Error(t, err)
}
