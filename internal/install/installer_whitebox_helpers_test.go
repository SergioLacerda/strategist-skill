package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// minimalExtractor creates the minimum .strategist/ layout needed by Install.
type minimalExtractor struct{}

func (m minimalExtractor) Extract(targetDir string, _ bool) error {
	dirs := []string{
		filepath.Join(targetDir, "personas"),
		filepath.Join(targetDir, "roles"),
		filepath.Join(targetDir, "templates"),
		filepath.Join(targetDir, "memory"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	files := map[string]string{
		filepath.Join(targetDir, "SKILL.md"):                               "# SKILL\n",
		filepath.Join(targetDir, "knowledge.index.yaml"):                   "sources: []\n",
		filepath.Join(targetDir, "treasure-chests.yaml"):                   "chests: []\n",
		filepath.Join(targetDir, "index.yaml"):                             "load_always: []\nload_by_task_type: {}\n",
		filepath.Join(targetDir, "templates", "pragmatic-standalone.yaml"): "mode: pragmatic\nbase_path: .analysis\n",
		filepath.Join(targetDir, "templates", "epic-standalone.yaml"):      "mode: epic\nbase_path: .analysis\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// minimalCatalogYAML is a valid, minimal plugins/catalog.yaml covering every
// provider id the whitebox wizard-flow tests script — kept in sync with
// internal/embed/defaults/plugins/catalog.yaml's risk_score values for the
// same ids. Since docs/adr/0035-embedded-weapon-fallback-policy.md,
// runWizard hard-blocks before any prompt when plugins/catalog.yaml fails to
// load, so minimalExtractor (used by every scripted wizard-completes-
// successfully test) must serve a loadable catalog rather than relying on
// the removed silent fallback. TestRunWizardBlocksOnUnreadableCatalog
// (wizard_test.go) covers the failure path directly, with its own extractor.
const minimalCatalogYAML = `schema_version: strategist-plugin-catalog/v1
providers:
  - id: archivist
    risk_score: write_analysis
    compatibility_source: native_role
  - id: batata
    risk_score: controlled
    compatibility_source: external
  - id: brainstorming
    risk_score: write_analysis
    category: discovery
    canonical_role: ranger
    default: true
    installable: true
    legacy_manifest_path: skills/brainstorming/skill.yaml
    compatibility_source: embedded
  - id: openspec-explore
    risk_score: write_analysis
    category: refinement
    canonical_role: archivist
    default: false
    installable: true
    legacy_manifest_path: skills/openspec-explore/skill.yaml
    compatibility_source: embedded
  - id: openspec-propose
    risk_score: write_analysis
    category: refinement
    canonical_role: archivist
    default: true
    installable: true
    legacy_manifest_path: skills/openspec-propose/skill.yaml
    compatibility_source: embedded
  - id: ranger
    risk_score: write_analysis
    compatibility_source: native_role
  - id: sdd-ask
    risk_score: controlled
    compatibility_source: external
  - id: sniper
    risk_score: controlled
    compatibility_source: native_role
`

func (m minimalExtractor) ReadFile(relPath string) ([]byte, error) {
	switch relPath {
	case "templates/epic-standalone.yaml":
		return []byte("mode: epic\nbase_path: .analysis\n"), nil
	case "SKILL.md":
		return []byte("# SKILL\n"), nil
	case pluginCatalogPath:
		return []byte(minimalCatalogYAML), nil
	case "roles/default.yaml":
		return []byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), nil
	case "roles/ranger.yaml":
		return []byte("role: ranger\nslot: discovery\n"), nil
	case "roles/archivist.yaml":
		return []byte("role: archivist\nslot: refinement\n"), nil
	case "roles/sniper.yaml":
		return []byte("role: sniper\nslot: execution\n"), nil
	case "skills/brainstorming/skill.yaml":
		return []byte("id: brainstorming\nstatus: active\nrisk_score: write_analysis\nprovider_class: rankeado\nspecialization_taxonomy:\n  canonical_role: ranger\n  provider_class: rankeado\nauxiliary_tools_allowed:\n  - writing-plans\n"), nil
	case "skills/openspec-explore/skill.yaml":
		return []byte("id: openspec-explore\nstatus: active\nrisk_score: write_analysis\n"), nil
	case "skills/openspec-propose/skill.yaml":
		return []byte("id: openspec-propose\nstatus: active\nrisk_score: write_analysis\n"), nil
	default:
		for _, file := range domain.NormativeRuntimeDefaultFiles() {
			if relPath == file.Path {
				return []byte(relPath + "\n"), nil
			}
		}
		return nil, fmt.Errorf("minimalExtractor: file not found: %s", relPath)
	}
}

type nopCompiler struct{}

func (nopCompiler) CompileAll(_, _ string) error { return nil }

func newSvcW(t *testing.T, wizardInput string) Service {
	t.Helper()
	return Service{
		Extractor:      minimalExtractor{},
		Compiler:       nopCompiler{},
		WizardPrompter: NewTextPrompter(strings.NewReader(wizardInput)),
		ShimHomeDir:    t.TempDir(),
	}
}

func p(input string) Prompter { return NewTextPrompter(strings.NewReader(input)) }
