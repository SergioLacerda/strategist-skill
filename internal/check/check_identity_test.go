package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckCmd_MissingIdentityFiles is the D6 regression guard: strategist check
// must fail loudly (not just preflight's soft degraded warn) when the internal
// domain identity files are absent from the .strategist/ root under check.
func TestCheckCmd_MissingIdentityFiles(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "templates", "domain", "identity")))

	checkRoot = dir
	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err, "check must fail when identity files are missing")
	assert.Contains(t, err.Error(), "check=blocked")
	assert.Contains(t, err.Error(), "reason=identity_files_missing")
	assert.Contains(t, err.Error(), "strategist compile")
}

// TestCheckCmd_MissingOneIdentityFile covers the partial case: only one of the
// two identity files absent must still be reported (both file names named).
func TestCheckCmd_MissingOneIdentityFile(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "templates", "domain", "identity", "what-i-am.yaml")))

	checkRoot = dir
	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err, "check must fail when even one identity file is missing")
	assert.Contains(t, err.Error(), "reason=identity_files_missing")
	assert.Contains(t, err.Error(), "identity/what-i-am.yaml")
}

// TestCheckCmd_IdentityFilesPresent is the happy path: minimalCheckRoot ships both
// identity files, so plain check must not fail on this account.
func TestCheckCmd_IdentityFilesPresent(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)

	checkRoot = dir
	err := checkCmd.RunE(checkCmd, nil)
	require.NoError(t, err, "check must pass when both identity files are present")
}
