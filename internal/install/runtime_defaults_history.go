package install

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// runtimeDefaultPolicy carries the operator's overrides for normative files.
type runtimeDefaultPolicy struct {
	Force          bool
	AllowDowngrade bool
}

// withInstallHistory carries the previous manifest's per-file install history
// into a manifest about to replace it, so a later run can recognize an older
// binary's defaults as a downgrade. An unreadable previous manifest yields no
// history rather than an error: it is about to be replaced, and the write
// itself reports any real I/O problem.
func withInstallHistory(strategistDir string, next domain.InstallManifest) domain.InstallManifest {
	prev, loaded, err := loadInstallManifest(strategistDir)
	if err != nil {
		return next
	}
	return next.WithHistoryFrom(prev, loaded)
}
