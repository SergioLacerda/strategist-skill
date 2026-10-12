package check

import (
	"path"
	"path/filepath"
	"strings"
)

// rankedRuntimeDir is the Strategist-root-relative directory holding one
// provider's materialized runtime.
func rankedRuntimeDir(provider string) (string, bool) {
	if provider == "" || provider == "." || provider == ".." || path.Base(provider) != provider || strings.ContainsAny(provider, `\:`) {
		return "", false
	}
	return "weapon-runtime/" + provider, true
}

// recordedScriptPath resolves the recorded OpenSpec launcher, which must be a
// canonical slash path inside weapon-runtime/<provider>/openspec/, so a
// tampered state file cannot point the healthcheck at another script.
func recordedScriptPath(root, provider, script string) (string, bool) {
	dir, ok := rankedRuntimeDir(provider)
	if !ok {
		return "", false
	}
	if !strings.HasPrefix(script, dir+"/openspec/") || path.Clean(script) != script || strings.ContainsAny(script, `\:`) {
		return "", false
	}
	return filepath.Join(root, filepath.FromSlash(script)), true
}
