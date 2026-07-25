package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// printContentByLang extracts personas.<persona>.content_by_lang.<lang> and, when present,
// personas.<persona>.phase_announcements.<lang> from the compiled config artifact and prints
// both as indented JSON to stdout. This is the supported way for an agent to read localized
// runtime message templates — persona YAML source under personas/<mode>.yaml only ever
// contains the canonical English content; non-English variants (e.g. pt-BR) are injected only
// at `strategist compile` time (see internal/compile/config.go's injectPTBRRuntime) and exist
// solely in the compiled artifact. content_by_lang is required — its absence or a missing lang
// key is an error. phase_announcements is optional — some personas (e.g. pragmatic) declare no
// phase_announcements at all, so its absence is not an error; when present but the requested
// lang key is missing, it is omitted from the output rather than failing the whole call.
func printContentByLang(root, persona, lang string) error {
	if persona == "" {
		return fmt.Errorf("[Strategist] check=blocked reason=persona_not_resolved\n→ pass --persona or ensure active.yaml has a non-empty mode")
	}

	artifact, err := readCompiledContentArtifact(root)
	if err != nil {
		return err
	}
	result, err := contentByLangOutput(artifact.Personas, persona, lang)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("[Strategist] check=blocked reason=marshal_error: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

type compiledContentArtifact struct {
	Personas map[string]any `json:"personas"`
}

func readCompiledContentArtifact(root string) (compiledContentArtifact, error) {
	artifactPath := filepath.Join(root, ".compiled", ".config.gz")
	f, err := os.Open(artifactPath) //nolint:gosec // G304: path derived from strategistDir
	if err != nil {
		return compiledContentArtifact{}, compiledArtifactOpenError(artifactPath, err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // read-only file; close error is not actionable

	gz, err := gzip.NewReader(f)
	if err != nil {
		return compiledContentArtifact{}, fmt.Errorf("[Strategist] check=blocked reason=compiled_artifact_corrupt: %w", err)
	}
	defer func() { _ = gz.Close() }() //nolint:errcheck // read-only reader; close error is not actionable

	var artifact compiledContentArtifact
	if err := json.NewDecoder(gz).Decode(&artifact); err != nil {
		return compiledContentArtifact{}, fmt.Errorf("[Strategist] check=blocked reason=compiled_artifact_corrupt: %w", err)
	}
	return artifact, nil
}

func compiledArtifactOpenError(path string, err error) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("[Strategist] check=blocked reason=compiled_artifact_missing artifact=%s\n→ Run: strategist compile", path)
	}
	return fmt.Errorf("[Strategist] check=blocked reason=compiled_artifact_read_error: %w", err)
}

func contentByLangOutput(personas map[string]any, persona, lang string) (map[string]any, error) {
	personaMap, err := compiledPersonaMap(personas, persona)
	if err != nil {
		return nil, err
	}
	langContent, err := requiredLangContent(personaMap, persona, lang)
	if err != nil {
		return nil, err
	}
	return addOptionalPhaseAnnouncements(map[string]any{"content_by_lang": langContent}, personaMap, lang), nil
}

func compiledPersonaMap(personas map[string]any, persona string) (map[string]any, error) {
	personaRaw, ok := personas[persona]
	if !ok {
		return nil, fmt.Errorf("[Strategist] check=blocked reason=persona_not_found persona=%s available=%s",
			persona, strings.Join(sortedKeys(personas), ","))
	}
	personaMap, ok := personaRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("[Strategist] check=blocked reason=persona_malformed persona=%s", persona)
	}
	return personaMap, nil
}

func requiredLangContent(personaMap map[string]any, persona, lang string) (any, error) {
	cbl, ok := personaMap["content_by_lang"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("[Strategist] check=blocked reason=content_by_lang_missing persona=%s", persona)
	}
	langContent, ok := cbl[lang]
	if !ok {
		return nil, fmt.Errorf("[Strategist] check=blocked reason=lang_not_found lang=%s persona=%s available=%s",
			lang, persona, strings.Join(sortedKeys(cbl), ","))
	}
	return langContent, nil
}

func addOptionalPhaseAnnouncements(result map[string]any, personaMap map[string]any, lang string) map[string]any {
	pa, ok := personaMap["phase_announcements"].(map[string]any)
	if !ok {
		return result
	}
	if paLang, ok := pa[lang]; ok {
		result["phase_announcements"] = paLang
	}
	return result
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
