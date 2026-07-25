// Package compile compiles Strategist skill artifacts from YAML sources into gzip-compressed JSON.
package compile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Compiler implements domain.Compiler.
type Compiler struct{}

// CompileAll runs Index, Domain, and Config in parallel and writes .manifest.gz
// only when all three succeed. All errors are collected and joined — a partial
// failure reports every failing step, not just the first.
func (c Compiler) CompileAll(root, indexPath string) error {
	compiledDir := filepath.Join(root, ".compiled")

	indexOut := filepath.Join(compiledDir, ".index.gz")
	domainOut := filepath.Join(compiledDir, ".domain.gz")
	configOut := filepath.Join(compiledDir, ".config.gz")
	manifestOut := filepath.Join(compiledDir, ".manifest.gz")

	steps := []struct {
		name string
		fn   func() error
	}{
		{"index", func() error { return Index(indexPath, indexOut) }},
		{"domain", func() error { return Domain(root, domainOut) }},
		{"config", func() error { return Config(root, configOut) }},
	}

	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)
	wg.Add(len(steps))
	for _, s := range steps {
		s := s
		go func() {
			defer wg.Done()
			if err := s.fn(); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("compile all: %s: %w", s.name, err))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(errs) > 0 {
		_ = os.Remove(manifestOut) //nolint:errcheck // best-effort: absence is the "partial compile" signal
		return errors.Join(errs...)
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

	if err := writeGzJSON(manifestOut, manifest); err != nil {
		return fmt.Errorf("compile all: manifest: %w", err)
	}

	return nil
}

// Ensure Compiler satisfies the domain interface at compile time.
var _ domain.Compiler = Compiler{}
