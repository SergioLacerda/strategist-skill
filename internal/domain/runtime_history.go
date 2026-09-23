package domain

// maxInstallHistory bounds InstallManifestFile.History so the manifest does
// not grow without limit across releases.
const maxInstallHistory = 16

// WithHistoryFrom carries each path's install history over from the previous
// manifest: when a path's hash changed, the previous hash becomes the most
// recent history entry. The current hash never appears in its own history.
func (m InstallManifest) WithHistoryFrom(prev InstallManifest, prevLoaded bool) InstallManifest {
	if !prevLoaded {
		return m
	}
	files := make([]InstallManifestFile, len(m.Files))
	for i, file := range m.Files {
		if old, ok := prev.FileByPath(file.Path); ok {
			file.History = mergeHistory(file.SHA256, old)
		}
		files[i] = file
	}
	m.Files = files
	return m
}

func mergeHistory(current string, previous InstallManifestFile) []string {
	candidates := previous.History
	if previous.SHA256 != current {
		candidates = append([]string{previous.SHA256}, previous.History...)
	}
	return boundedUniqueHashes(candidates, current)
}

// boundedUniqueHashes keeps the first occurrence of each non-empty hash other
// than exclude, in order, up to maxInstallHistory entries.
func boundedUniqueHashes(hashes []string, exclude string) []string {
	var out []string
	seen := map[string]bool{exclude: true, "": true}
	for _, hash := range hashes {
		if !seen[hash] && len(out) < maxInstallHistory {
			seen[hash] = true
			out = append(out, hash)
		}
	}
	return out
}

func containsHash(hashes []string, hash string) bool {
	for _, h := range hashes {
		if h == hash {
			return true
		}
	}
	return false
}

// isDowngrade reports a runtime that still holds its last installed default
// while the running binary carries a default that runtime was installed with
// earlier: the binary is older than the runtime.
func isDowngrade(in RuntimeDefaultDecisionInput) bool {
	return !in.AllowDowngrade && in.Exists && in.HasManifest &&
		in.CurrentHash == in.ManifestHash && in.EmbeddedHash != in.CurrentHash &&
		containsHash(in.ManifestHistory, in.EmbeddedHash)
}
