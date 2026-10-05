package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateProjectRelativeDirectory(t *testing.T) {
	for _, value := range []string{"adr", ".analysis/adr", "docs\\runbooks"} {
		require.NoError(t, ValidateProjectRelativeDirectory(value, "root"), value)
	}
	for _, value := range []string{"/tmp/adr", "../adr", "..", "."} {
		require.Error(t, ValidateProjectRelativeDirectory(value, "root"), value)
	}
}
