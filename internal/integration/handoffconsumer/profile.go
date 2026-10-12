package handoffconsumer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
)

// ProfileID names the default question profile of the handoff pre-check: one
// aggregate accept or reject choice plus one noul per criterion. A calibration is
// bound to a profile; a changed profile is a different judge.
const ProfileID = calibration.ProfileV1

// TransitionForBoundary maps a calibration boundary to the consuming transition
// whose criteria it exercises.
func TransitionForBoundary(boundary string) string {
	if boundary == calibration.BoundaryRanger {
		return handoff.TransitionRangerToArchivist
	}
	return TransitionArchivistToGate
}

// ProfileVersion is the version of the default profile at a boundary.
func ProfileVersion(boundary string) string { return ProfileVersionOf(ProfileID, boundary) }

// ProfileVersionOf is a digest of the exact questions a profile asks at a
// boundary (and, for v2, of its projection rule), so any change to a criterion's
// wording or to how absent fields are shown is a new version and invalidates the
// calibrations bound to the old one. The v1 digest is the questions alone.
func ProfileVersionOf(profile, boundary string) string {
	raw, err := json.Marshal(QuestionsFor(profile, TransitionForBoundary(boundary)))
	if err != nil {
		return "unknown"
	}
	if calibration.ShowsAbsent(calibration.ResolveProfile(profile)) {
		raw = append(raw, []byte("|"+absentRule)...)
	}
	if calibration.IsAtomic(profile) {
		raw = append(raw, []byte("|"+combineRule)...)
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))[:8]
}

// Binding is the calibration binding of a call at a boundary and resolved model, for the default profile.
func Binding(boundary, model string) calibration.Binding {
	return BindingFor(ProfileID, boundary, model)
}

// BindingFor is the calibration binding for a profile.
func BindingFor(profile, boundary, model string) calibration.Binding {
	profile = calibration.ResolveProfile(profile)
	return calibration.Binding{Model: model, Profile: profile, ProfileVersion: ProfileVersionOf(profile, boundary)}
}
