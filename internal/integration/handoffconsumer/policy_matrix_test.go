package handoffconsumer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThePolicyMatrixHoldsAndNamesEveryRule(t *testing.T) {
	rows := PolicyMatrix(time.Now())
	require.GreaterOrEqual(t, len(rows), 25)
	names := map[string]bool{}
	for _, r := range rows {
		assert.True(t, r.Pass, "%s: %s", r.Name, r.Detail)
		assert.False(t, names[r.Name], "duplicate row %s", r.Name)
		names[r.Name] = true
	}
	assert.True(t, names["policy never approves at exactly the threshold"])
	assert.True(t, names["policy flags a binding integrity failure as suspect"])
	assert.True(t, names["pinned calibration is ignored for another model"])
}
