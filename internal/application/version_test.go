package application_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestDisplayVersion(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		raw, want string
	}{
		{raw: "1.0.18", want: "V1.0.18"},
		{raw: "v1.0.18-2-gabcdef0", want: "V1.0.18+"},
		{raw: "dev", want: "Vdev"},
	} {
		t.Run(test.raw, func(t *testing.T) {
			require.Equal(t, test.want, application.DisplayVersion(test.raw))
		})
	}
}
