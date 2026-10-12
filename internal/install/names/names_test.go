package names

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultShimPathIsUnderTheAgentHomeSkillsDirectory(t *testing.T) {
	t.Parallel()
	home := filepath.Join("home", "dev")
	assert.Equal(t, filepath.Join(home, ".claude", "skills", "strategist", "SKILL.md"), DefaultShimPath(home))
	assert.Equal(t, filepath.Join("skills", "strategist", "SKILL.md"), ShimRelPath)
}
