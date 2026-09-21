//go:build strategist_payload

package main

// Registers the runtime payload embedded in this build (see
// internal/runtimepayload/bundled). Ordinary builds do not import it.
import _ "github.com/SergioLacerda/strategist-skill/internal/runtimepayload/bundled"
