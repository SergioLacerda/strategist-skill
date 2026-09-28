package check

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// layoutSkewAdvisories reports a runtime whose install manifest records a newer
// layout generation than this binary writes (domain.RuntimeLayoutGeneration). It is
// non-blocking: the binary trusts its own layout, so it keeps resolving what it
// knows, and the operator is told to update the binary instead of reinstalling.
// This is a different signal from the blocking, hash-based runtime_newer_than_binary
// (see runtime_defaults.go): that one compares normative file hashes, this one
// compares the layout marker. An absent manifest, an absent field or an unreadable
// manifest is generation 0 and never produces a warning here.
func layoutSkewAdvisories(root string) []string {
	manifest, found, err := readInstallManifest(root)
	if err != nil || !found || manifest.RuntimeLayoutGeneration <= domain.RuntimeLayoutGeneration {
		return nil
	}
	return []string{fmt.Sprintf(
		"[Strategist] phase=preflight status=warn reason=runtime_layout_newer_than_binary runtime_generation=%d binary_generation=%d (the runtime was laid out by a newer binary; this binary trusts its own layout — update the binary, do not reinstall; see docs/runbooks/embedded-defaults-drift-recovery.md)",
		manifest.RuntimeLayoutGeneration, domain.RuntimeLayoutGeneration)}
}
