// Package compile compiles Strategist skill artifacts from YAML sources into gzip-compressed JSON.
package compile

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Compiler implements domain.Compiler.
type Compiler struct{}

// CompileAll orchestrates Index, Domain, and Config in sequence,
// writing .manifest.gz only when all three succeed.
// It mirrors the logic of compile-all.sh exactly.
func (c Compiler) CompileAll(root, indexPath string) error {
	compiledDir := filepath.Join(root, ".compiled")

	indexOut := filepath.Join(compiledDir, ".index.gz")
	if err := Index(indexPath, indexOut); err != nil {
		return fmt.Errorf("compile all: index: %w", err)
	}

	domainOut := filepath.Join(compiledDir, ".domain.gz")
	if err := Domain(root, domainOut); err != nil {
		return fmt.Errorf("compile all: domain: %w", err)
	}

	configOut := filepath.Join(compiledDir, ".config.gz")
	if err := Config(root, configOut); err != nil {
		return fmt.Errorf("compile all: config: %w", err)
	}

	manifest := compiledManifest{
		Schema:      "strategist-compiled-manifest/1.0",
		GeneratedAt: time.Now().Unix(),
		Artifacts: map[string]string{
			".index.gz":  sha256Artifact(indexOut),
			".domain.gz": sha256Artifact(domainOut),
			".config.gz": sha256Artifact(configOut),
		},
	}

	manifestOut := filepath.Join(compiledDir, ".manifest.gz")
	if err := writeGzJSON(manifestOut, manifest); err != nil {
		return fmt.Errorf("compile all: manifest: %w", err)
	}

	return nil
}

// Ensure Compiler satisfies the domain interface at compile time.
var _ domain.Compiler = Compiler{}
