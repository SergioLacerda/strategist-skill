//go:build strategist_payload

package bundled

import (
	"io/fs"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
)

// registerTarget is called from the per-target embed files (built only with
// -tags strategist_payload, after scripts/fetch-node-runtime.py). A payload that
// cannot be composed is a defective build, so it stops the program at start-up
// instead of silently falling back to a host executable.
func registerTarget(target string, nodeFS fs.FS) {
	if err := runtimepayload.RegisterEmbedded(embed.DefaultsFS(), nodeFS, target); err != nil {
		panic(runtimepayload.InitFailureMessage(target, err))
	}
}
