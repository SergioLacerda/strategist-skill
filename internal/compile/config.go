package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

// Config reads active.yaml, personas/*.yaml and roles/*.yaml from root
// and writes a gzip-compressed JSON artifact to outputPath.
// It mirrors the logic of compile-config.sh exactly.
func Config(root, outputPath string) error {
	activePath := filepath.Join(root, "active.yaml")
	activeRaw, err := loadValidatedActiveRaw(activePath)
	if err != nil {
		return fmt.Errorf("compile config: %w", err)
	}

	sources := map[string]int64{activePath: mtime(activePath)}

	personasRaw, err := compilePersonas(root, sources)
	if err != nil {
		return err
	}

	roles, err := compileYAMLDirTyped[map[string]any](filepath.Join(root, "roles"), sources)
	if err != nil {
		return fmt.Errorf("compile config: roles: %w", err)
	}

	artifact := compiledConfig{
		Schema:     "strategist-compiled-config/1.0",
		CompiledAt: time.Now().Unix(),
		Sources:    sources,
		Active:     activeRaw,
		Personas:   mapValuesToAny(personasRaw),
		Roles:      mapValuesToAny(roles),
	}

	if err := writeGzJSON(outputPath, artifact); err != nil {
		return fmt.Errorf("compile config: write %s: %w", outputPath, err)
	}
	return nil
}

func compilePersonas(root string, sources map[string]int64) (map[string]any, error) {
	personasDir := filepath.Join(root, "personas")
	personasRaw, err := compileYAMLDirTyped[map[string]any](personasDir, sources)
	if err != nil {
		return nil, fmt.Errorf("compile config: personas: %w", err)
	}
	if err := validateTypedPersonas(personasDir); err != nil {
		return nil, err
	}
	injectPTBRRuntime(personasRaw)
	return mapValuesToAny(personasRaw), nil
}

func mapValuesToAny[T any](values map[string]T) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func validateTypedPersonas(personasDir string) error {
	personasTyped, err := compileYAMLDirTyped[PersonaConfig](personasDir, nil)
	if err != nil {
		return fmt.Errorf("compile config: personas validate: %w", err)
	}
	for name, persona := range personasTyped {
		if err := persona.Validate(); err != nil {
			return fmt.Errorf("compile config: personas/%s: %w", name, err)
		}
	}
	return nil
}

func injectPTBRRuntime(personasRaw map[string]map[string]any) {
	ptBRRuntime, _ := i18n.RuntimeBundleFor(i18n.LangPTBR)
	ptBRPhaseAnnouncements, _ := i18n.PhaseAnnouncementsFor(i18n.LangPTBR)
	for _, raw := range personasRaw {
		if cbl, ok := contentByLang(raw); ok {
			cbl[i18n.LangPTBR] = ptBRRuntime.ToMap()
		}
		if pa, ok := phaseAnnouncements(raw); ok {
			pa[i18n.LangPTBR] = ptBRPhaseAnnouncements.ToMap()
		}
	}
}

func contentByLang(raw any) (map[string]any, bool) {
	personaMap, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	cbl, ok := personaMap["content_by_lang"].(map[string]any)
	return cbl, ok
}

func phaseAnnouncements(raw any) (map[string]any, bool) {
	personaMap, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	pa, ok := personaMap["phase_announcements"].(map[string]any)
	return pa, ok
}

func loadValidatedActiveRaw(activePath string) (map[string]any, error) {
	activeData, err := os.ReadFile(activePath) //nolint:gosec // G304: path derived from strategistDir
	if err != nil {
		return nil, fmt.Errorf("read active.yaml: %w", err)
	}

	// Validate active.yaml using a typed struct (catches required-field drift early).
	var activeCfg ActiveConfig
	if err := parseYAMLBytes(activePath, activeData, &activeCfg); err != nil {
		return nil, err
	}
	if err := activeCfg.Validate(); err != nil {
		return nil, fmt.Errorf("active.yaml: %w", err)
	}

	// Load as raw map so all fields (including extended ones like treasure_chests) are preserved.
	activeRaw, err := parseYAMLMapBytes(activePath, activeData)
	if err != nil {
		return nil, fmt.Errorf("active.yaml raw: %w", err)
	}
	// roles_config was a transitional slot-map pointer. Runtime resolution now
	// reads active.slots directly, so do not carry the stale field into compiled
	// artifacts even when old installs still have it in active.yaml.
	delete(activeRaw, "roles_config")
	return activeRaw, nil
}

// compileYAMLDirTyped reads all *.yaml files from dir and returns a typed map
// keyed by basename-without-ext.
func compileYAMLDirTyped[T any](dir string, sources map[string]int64) (map[string]T, error) {
	entries, missing, err := readYAMLDirEntries(dir)
	if missing {
		return map[string]T{}, nil
	}
	if err != nil {
		return nil, err
	}

	result := make(map[string]T, len(entries))
	for _, e := range entries {
		if !isYAMLFile(e) {
			continue
		}
		if err := addTypedYAMLFile(result, sources, dir, e.Name()); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func readYAMLDirEntries(dir string) ([]os.DirEntry, bool, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read dir %s: %w", dir, err)
	}
	return entries, false, nil
}

func addTypedYAMLFile[T any](result map[string]T, sources map[string]int64, dir, name string) error {
	path := filepath.Join(dir, name)
	content, err := loadTypedYAMLFile[T](path)
	if err != nil {
		return err
	}
	if sources != nil {
		sources[path] = mtime(path)
	}
	result[strings.TrimSuffix(name, ".yaml")] = content
	return nil
}

func isYAMLFile(e os.DirEntry) bool {
	return !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml")
}

func loadTypedYAMLFile[T any](path string) (T, error) {
	var content T
	if err := loadYAMLInto(path, &content); err != nil {
		return content, err
	}
	return content, nil
}
