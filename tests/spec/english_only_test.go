//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStrategistSourceTreeEnglishOnly scans strategist/ for Portuguese prose markers.
//
// Allowlisted paths and fields that legitimately contain non-English data:
//   - strategist/schemas/intake.schema.yaml — user input aliases (não pode quebrar, etc.)
//     are intent-matching tokens, not prose (design non-goal: preserve Portuguese input tokens).
//   - strategist/contracts/machine/adr.yaml — pt-BR section name list (data).
//   - strategist/contracts/narrative/07-adr.md — pt-BR language mapping (data).
//   - strategist/contracts/adr.md — docs: pt-BR language mapping (data).
//   - strategist/contracts/machine/critical-hit.yaml — reserved input tokens with inline doc.
func TestStrategistSourceTreeEnglishOnly(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	strategistDir := filepath.Join(root, "internal", "embed", "defaults")

	// Portuguese prose markers that must NOT appear in canonical strategist/ files.
	// These are not language-code data — they are prose fragments in Portuguese.
	forbiddenProse := []string{
		"Não processe",
		"execute antes de qualquer coisa",
		"execute exatamente nessa ordem",
		"fluxo completo",
		"fluxo direto",
		"orquestração em lote",
		"análises capturadas",
		"Caminho para arquivo",
		"processar todas",
		"pacotes de negócio",
		"camada pura",
		"ponto de wiring",
		"Nunca executar",
		"Nunca ler de",
		"Nunca pular",
		"Nunca invocar",
		"Missão: {mission_id}",
		"Perfil: {profile}",
		"Arquivista →",
		"aprovação concedida",
		"reconhecimento concluído",
		"implementação concluída",
		"Autorizar Sniper",
		"Aguardando confirmação",
		"Commitar?",
	}

	err := filepath.WalkDir(strategistDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".md" && ext != ".yaml" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		content := string(data)
		for _, marker := range forbiddenProse {
			if strings.Contains(content, marker) {
				t.Errorf("strategist/ prose violation: %s contains Portuguese prose marker %q", path, marker)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk strategist/: %v", err)
	}
}
