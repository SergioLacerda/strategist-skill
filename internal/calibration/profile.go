package calibration

// Question profiles of the handoff pre-check. A calibration is bound to one: a
// different profile asks different questions or projects differently, so it is a
// different judge.
const (
	// ProfileV1 projects only the fields the artifact has.
	ProfileV1 = "handoff-v1"
	// ProfileV2 also shows a catalog field the artifact lacks as `(absent)`, so the
	// provider can judge what is missing instead of never seeing it.
	ProfileV2 = "handoff-v2"
	// ProfileAtomicV1 asks one noul question per criterion and no aggregate choice;
	// the probability of accept is the lowest of the criterion scores. It shows an
	// absent field like v2. It exists to be measured in the DOJO: the handoff does
	// not run it until a calibration of it is adopted.
	ProfileAtomicV1 = "handoff-atomic-v1"
)

// KnownProfile reports whether id names a profile the handoff pre-check can run;
// the empty id is the default (v1).
func KnownProfile(id string) bool { return id == "" || id == ProfileV1 || id == ProfileV2 }

// KnownCalibrationProfile reports whether id names a profile a DOJO case can measure.
func KnownCalibrationProfile(id string) bool { return KnownProfile(id) || id == ProfileAtomicV1 }

// IsAtomic reports whether the profile asks one question per criterion.
func IsAtomic(id string) bool { return id == ProfileAtomicV1 }

// ShowsAbsent reports whether the profile shows a missing catalog field as `(absent)`.
func ShowsAbsent(id string) bool { return id == ProfileV2 || id == ProfileAtomicV1 }

// ResolveProfile maps the empty id to the default profile.
func ResolveProfile(id string) string {
	if id == "" {
		return ProfileV1
	}
	return id
}
