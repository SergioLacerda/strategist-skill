//go:build strategist_payload

package runtimepayload

import (
	"fmt"
	"io/fs"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
)

// registerTarget is called from the per-target embed files (built only with
// -tags strategist_payload, after scripts/fetch-node-runtime.py). A payload
// that cannot be composed is a defective build, so it stops the program at
// start-up instead of silently falling back to a host executable.
func registerTarget(target string, nodeFS fs.FS) {
	m, src, err := BuildEmbedded(embed.DefaultsFS(), nodeFS, target)
	if err != nil {
		panic(fmt.Sprintf("strategist: embedded runtime payload for %s is invalid: %v", target, err))
	}
	Register(m, src)
}
