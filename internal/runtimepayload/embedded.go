package runtimepayload

import "io/fs"

var embeddedOpenSpec fs.FS

// RegisterOpenSpec supplies the digest-verified OpenSpec defaults used by the
// installer. The CLI registers internal/embed.DefaultsFS during startup; the
// package remains independent from the embed layer so its tests and consumers
// do not create an import cycle.
func RegisterOpenSpec(src fs.FS) { embeddedOpenSpec = src }

// DefaultOpenSpec returns the registered embedded OpenSpec source.
func DefaultOpenSpec() (fs.FS, bool) {
	if embeddedOpenSpec == nil {
		return nil, false
	}
	return embeddedOpenSpec, true
}
