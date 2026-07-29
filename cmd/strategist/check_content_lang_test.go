package main

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCmd_PrintContentByLang_Success(t *testing.T) {
	dir := minimalCheckRoot(t)
	testutil.WriteGzJSON(t, filepath.Join(dir, ".compiled", ".config.gz"), map[string]any{
		"personas": map[string]any{
			"epic": map[string]any{
				"content_by_lang": map[string]any{
					"en":    map[string]any{"ranger_start": "starting"},
					"pt-BR": map[string]any{"ranger_start": "iniciando"},
				},
			},
		},
	})

	orig, origPersona := checkRoot, checkPrintContentByLangPersona
	t.Cleanup(func() {
		checkRoot, checkPrintContentByLangPersona = orig, origPersona
		checkPrintContentByLang = ""
	})
	checkRoot = dir
	checkPrintContentByLang = "pt-BR"

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = checkCmd.RunE(checkCmd, nil)
	})
	require.NoError(t, runErr)
	assert.Contains(t, stdout, "iniciando")
	assert.NotContains(t, stdout, "starting")
}

func TestCheckCmd_PrintContentByLang_DefaultsPersonaFromActiveYAML(t *testing.T) {
	dir := minimalCheckRoot(t)
	testutil.WriteGzJSON(t, filepath.Join(dir, ".compiled", ".config.gz"), map[string]any{
		"personas": map[string]any{
			"epic": map[string]any{
				"content_by_lang": map[string]any{
					"en": map[string]any{"ranger_start": "starting"},
				},
			},
		},
	})

	orig := checkRoot
	t.Cleanup(func() {
		checkRoot = orig
		checkPrintContentByLang = ""
	})
	checkRoot = dir
	checkPrintContentByLang = "en"
	// checkPrintContentByLangPersona left empty — must default to active.yaml's mode (epic).

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = checkCmd.RunE(checkCmd, nil)
	})
	require.NoError(t, runErr)
	assert.Contains(t, stdout, "starting")
}

func TestCheckCmd_PrintContentByLang_MissingCompiledArtifact(t *testing.T) {
	dir := minimalCheckRoot(t)

	orig := checkRoot
	t.Cleanup(func() {
		checkRoot = orig
		checkPrintContentByLang = ""
	})
	checkRoot = dir
	checkPrintContentByLang = "pt-BR"

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compiled_artifact_missing")
	assert.Contains(t, err.Error(), "strategist compile")
}

func TestCheckCmd_PrintContentByLang_MissingLangKey(t *testing.T) {
	dir := minimalCheckRoot(t)
	testutil.WriteGzJSON(t, filepath.Join(dir, ".compiled", ".config.gz"), map[string]any{
		"personas": map[string]any{
			"epic": map[string]any{
				"content_by_lang": map[string]any{
					"en": map[string]any{"ranger_start": "starting"},
				},
			},
		},
	})

	orig := checkRoot
	t.Cleanup(func() {
		checkRoot = orig
		checkPrintContentByLang = ""
	})
	checkRoot = dir
	checkPrintContentByLang = "xx-XX"

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lang_not_found")
	assert.Contains(t, err.Error(), "available=en")
}

func TestCheckCmd_PrintContentByLang_MissingPersona(t *testing.T) {
	dir := minimalCheckRoot(t)
	testutil.WriteGzJSON(t, filepath.Join(dir, ".compiled", ".config.gz"), map[string]any{
		"personas": map[string]any{
			"epic": map[string]any{
				"content_by_lang": map[string]any{"en": map[string]any{}},
			},
		},
	})

	orig, origPersona := checkRoot, checkPrintContentByLangPersona
	t.Cleanup(func() {
		checkRoot, checkPrintContentByLangPersona = orig, origPersona
		checkPrintContentByLang = ""
	})
	checkRoot = dir
	checkPrintContentByLang = "en"
	checkPrintContentByLangPersona = "pragmatic"

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persona_not_found")
	assert.Contains(t, err.Error(), "available=epic")
}

func TestCheckCmd_PrintContentByLang_IncludesPhaseAnnouncementsWhenPresent(t *testing.T) {
	dir := minimalCheckRoot(t)
	testutil.WriteGzJSON(t, filepath.Join(dir, ".compiled", ".config.gz"), map[string]any{
		"personas": map[string]any{
			"epic": map[string]any{
				"content_by_lang": map[string]any{
					"en":    map[string]any{"ranger_start": "starting"},
					"pt-BR": map[string]any{"ranger_start": "iniciando"},
				},
				"phase_announcements": map[string]any{
					"en":    map[string]any{"discovery_starting": "Ranger starts."},
					"pt-BR": map[string]any{"discovery_starting": "Ranger começa."},
				},
			},
		},
	})

	orig := checkRoot
	t.Cleanup(func() {
		checkRoot = orig
		checkPrintContentByLang = ""
	})
	checkRoot = dir
	checkPrintContentByLang = "pt-BR"

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = checkCmd.RunE(checkCmd, nil)
	})
	require.NoError(t, runErr)
	assert.Contains(t, stdout, "Ranger começa.")
	assert.NotContains(t, stdout, "Ranger starts.")
}

func TestCheckCmd_PrintContentByLang_OmitsPhaseAnnouncementsWhenAbsent(t *testing.T) {
	dir := minimalCheckRoot(t)
	testutil.WriteGzJSON(t, filepath.Join(dir, ".compiled", ".config.gz"), map[string]any{
		"personas": map[string]any{
			"epic": map[string]any{
				"content_by_lang": map[string]any{
					"en": map[string]any{"ranger_start": "starting"},
				},
			},
		},
	})

	orig := checkRoot
	t.Cleanup(func() {
		checkRoot = orig
		checkPrintContentByLang = ""
	})
	checkRoot = dir
	checkPrintContentByLang = "en"

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = checkCmd.RunE(checkCmd, nil)
	})
	require.NoError(t, runErr)
	assert.Contains(t, stdout, "content_by_lang")
	assert.NotContains(t, stdout, "phase_announcements")
}

func TestPrintContentByLang_ErrorBranches(t *testing.T) {
	t.Run("persona malformed", func(t *testing.T) {
		dir := minimalCheckRoot(t)
		testutil.WriteGzJSON(t, filepath.Join(dir, ".compiled", ".config.gz"), map[string]any{
			"personas": map[string]any{"epic": "not-a-map"},
		})

		err := printContentByLang(dir, "epic", "en")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "persona_malformed")
	})

	t.Run("content by lang missing", func(t *testing.T) {
		dir := minimalCheckRoot(t)
		testutil.WriteGzJSON(t, filepath.Join(dir, ".compiled", ".config.gz"), map[string]any{
			"personas": map[string]any{"epic": map[string]any{}},
		})

		err := printContentByLang(dir, "epic", "en")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "content_by_lang_missing")
	})

	t.Run("corrupt gzip", func(t *testing.T) {
		dir := minimalCheckRoot(t)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".compiled"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".compiled", ".config.gz"), []byte("not gzip"), 0o644))

		err := printContentByLang(dir, "epic", "en")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compiled_artifact_corrupt")
	})

	t.Run("directory artifact is corrupt", func(t *testing.T) {
		dir := minimalCheckRoot(t)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".compiled"), 0o755))
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".compiled", ".config.gz"), 0o755))

		err := printContentByLang(dir, "epic", "en")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compiled_artifact_corrupt")
	})

	t.Run("empty persona", func(t *testing.T) {
		dir := minimalCheckRoot(t)
		err := printContentByLang(dir, "", "en")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "persona_not_resolved")
	})

	t.Run("valid gzip with invalid JSON payload", func(t *testing.T) {
		dir := minimalCheckRoot(t)
		artifactPath := filepath.Join(dir, ".compiled", ".config.gz")
		require.NoError(t, os.MkdirAll(filepath.Dir(artifactPath), 0o755))
		f, err := os.Create(artifactPath)
		require.NoError(t, err)
		gz := gzip.NewWriter(f)
		_, err = gz.Write([]byte("not json"))
		require.NoError(t, err)
		require.NoError(t, gz.Close())
		require.NoError(t, f.Close())

		err = printContentByLang(dir, "epic", "en")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compiled_artifact_corrupt")
	})

	t.Run("artifact unreadable due to permissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("chmod not reliable on windows")
		}
		dir := minimalCheckRoot(t)
		artifactPath := filepath.Join(dir, ".compiled", ".config.gz")
		testutil.WriteGzJSON(t, artifactPath, map[string]any{"personas": map[string]any{}})
		require.NoError(t, os.Chmod(artifactPath, 0o000))
		t.Cleanup(func() { _ = os.Chmod(artifactPath, 0o644) })
		if os.Getuid() == 0 {
			t.Skip("running as root — file permission checks do not apply")
		}

		err := printContentByLang(dir, "epic", "en")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compiled_artifact_read_error")
	})
}
