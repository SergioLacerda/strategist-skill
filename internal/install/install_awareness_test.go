package install_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall_AwarenessRefresherCalledAfterInstall(t *testing.T) {
	t.Parallel()

	t.Run("silent install calls refresher", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		called := false
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.AwarenessRefresher = func(strategistRoot, projectRoot, _ string) bool {
			called = true
			assert.NotEmpty(t, strategistRoot)
			assert.NotEmpty(t, projectRoot)
			return true
		}
		require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
		assert.True(t, called, "AwarenessRefresher must be called after silent install")
	})

	t.Run("nil refresher does not panic", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.AwarenessRefresher = nil
		require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
	})

	t.Run("refresher returning false is non-fatal", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.AwarenessRefresher = func(_, _, _ string) bool { return false }
		err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
		require.NoError(t, err, "partial awareness failure must not abort install")
	})

	t.Run("refresher receives correct strategistRoot and projectRoot", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		var gotStrategistRoot, gotProjectRoot string
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.AwarenessRefresher = func(strategistRoot, projectRoot, _ string) bool {
			gotStrategistRoot = strategistRoot
			gotProjectRoot = projectRoot
			return true
		}
		require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
		assert.Equal(t, filepath.Join(dir, ".strategist"), gotStrategistRoot)
		assert.Equal(t, dir, gotProjectRoot)
	})

	t.Run("refresher receives version from Service.Version", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		var gotVersion string
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.Version = "1.2.3"
		svc.AwarenessRefresher = func(_, _, version string) bool {
			gotVersion = version
			return true
		}
		require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
		assert.Equal(t, "1.2.3", gotVersion)
	})
}
