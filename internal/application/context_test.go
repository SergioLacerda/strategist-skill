package application_test

import (
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

type contextReader map[string][]byte

func (r contextReader) ReadFile(ref string) ([]byte, error) { return r[ref], nil }

func TestMaterializeMissionContextLoadsMissionBeforeReadingContext(t *testing.T) {
	t.Parallel()

	loaded := false
	result, err := application.MaterializeMissionContext(
		func(_, _ string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
			loaded = true
			return nil, domain.MissionEngineStatus{MissionID: "m-1"}, nil
		},
		"root", "m-1", contextReader{"brief.md": []byte("brief")},
		[]domain.ContextReference{{Ref: "brief.md", Kind: "context"}}, 1, 100,
	)

	require.NoError(t, err)
	require.True(t, loaded)
	require.Equal(t, 5, result.Bytes)
}

func TestMaterializeMissionContextPropagatesMissionLoadError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("mission missing")
	_, err := application.MaterializeMissionContext(
		func(_, _ string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
			return nil, domain.MissionEngineStatus{}, wantErr
		}, "root", "m-1", contextReader{}, nil, 1, 100,
	)

	require.ErrorIs(t, err, wantErr)
}

func TestMaterializeMissionContextRejectsMissingLoader(t *testing.T) {
	t.Parallel()

	_, err := application.MaterializeMissionContext(nil, "root", "m-1", contextReader{}, nil, 1, 100)
	require.EqualError(t, err, "mission context: mission loader is required")
}
