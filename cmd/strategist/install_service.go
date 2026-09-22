package main

import (
	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/install"
)

// installExtractorOverride is a test-only seam: nil in production (real
// embedded default tree always used), settable by tests that only need
// some installed tree to exist rather than the full ~170-file/~3MB
// fidelity, to skip that extraction/compile/ranked-bootstrap cost. See
// docs/runbooks/test-scan-scope-hygiene.md.
var installExtractorOverride domain.FileExtractor

// installService is defined here, alongside installExtractorOverride, split
// out of install.go to keep that file under the repo's file-size budget.
func installService(shimHome string) install.Service {
	svc := install.Service{
		Extractor:          embedpkg.Extractor{},
		Lister:             embedpkg.Extractor{},
		Compiler:           compile.Compiler{},
		ShimHomeDir:        shimHome,
		AwarenessRefresher: refreshAgentAwarenessFromEmbed,
		Version:            Version,
	}
	if installExtractorOverride != nil {
		svc.Extractor = installExtractorOverride
		// extractRuntimeTree (internal/install/merge_extract.go) only runs the
		// three-way merge/backup path when Lister is non-nil; a Lister-capable
		// override opts into that (needed by force-overwrite/backup tests),
		// an Extractor-only override falls back to the simpler legacy path.
		if lister, ok := installExtractorOverride.(domain.FileLister); ok {
			svc.Lister = lister
		} else {
			svc.Lister = nil
		}
	}
	return svc
}
