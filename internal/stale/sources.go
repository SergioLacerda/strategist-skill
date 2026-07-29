package stale

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

type artifactSources struct {
	legacySources map[string]int64
	strongSources map[string]SourceMetadata
	hasLegacy     bool
	hasStrong     bool
}

func checkArtifactSources(result Result, sources artifactSources) (Result, error) {
	if sources.hasStrong {
		return checkStrongSources(result, sources.strongSources)
	}
	if sources.hasLegacy {
		result.Detail = "legacy sources metadata"
		return checkLegacySources(result, sources.legacySources)
	}
	result.Detail = "missing source metadata"
	return result, nil
}

func checkStrongSources(result Result, sources map[string]SourceMetadata) (Result, error) {
	for path, recorded := range sources {
		reason, staleSource, err := checkStrongSource(path, recorded)
		if err != nil {
			return result, err
		}
		if reason != ReasonFresh {
			result.Stale = true
			result.Reason = reason
			result.SourcePath = staleSource
			return result, nil
		}
	}
	return result, nil
}

func checkStrongSource(path string, recorded SourceMetadata) (Reason, string, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return ReasonMissingSource, path, nil
	}
	if err != nil {
		return ReasonFresh, "", fmt.Errorf("stale: stat source %s: %w", path, err)
	}
	if sourceMTimeNewer(info, recorded) {
		return ReasonSourceNewer, path, nil
	}
	if info.Size() != recorded.Size {
		return ReasonSourceMetadataMismatch, path, nil
	}
	if recorded.SHA256 != "" && sourceSHA256(path) != recorded.SHA256 {
		return ReasonSourceMetadataMismatch, path, nil
	}
	return ReasonFresh, "", nil
}

func sourceMTimeNewer(info os.FileInfo, recorded SourceMetadata) bool {
	if recorded.MTimeNS != 0 {
		return info.ModTime().UnixNano() > recorded.MTimeNS
	}
	return info.ModTime().Unix() > recorded.MTime
}

func checkLegacySources(result Result, sources map[string]int64) (Result, error) {
	for path, recorded := range sources {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			result.Stale = true
			result.Reason = ReasonMissingSource
			result.SourcePath = path
			return result, nil
		}
		if err != nil {
			return result, fmt.Errorf("stale: stat source %s: %w", path, err)
		}
		if info.ModTime().Unix() > recorded {
			result.Stale = true
			result.Reason = ReasonSourceNewer
			result.SourcePath = path
			return result, nil
		}
	}
	return result, nil
}

func sourceSHA256(path string) string {
	hash, exists, err := runtimefs.ReadSHA256(path)
	if err != nil || !exists {
		return ""
	}
	return hash
}

func readArtifactSources(artifactPath string) (artifactSources, error) {
	var raw map[string]json.RawMessage
	if err := readGzJSON(artifactPath, &raw); err != nil {
		return artifactSources{}, err
	}

	var out artifactSources
	if data, ok := raw["source_stats"]; ok {
		if err := json.Unmarshal(data, &out.strongSources); err != nil {
			return artifactSources{}, fmt.Errorf("source_stats: %w", err)
		}
		out.hasStrong = true
	}
	if data, ok := raw["sources"]; ok {
		if err := json.Unmarshal(data, &out.legacySources); err != nil {
			return artifactSources{}, fmt.Errorf("sources: %w", err)
		}
		out.hasLegacy = true
	}
	return out, nil
}
