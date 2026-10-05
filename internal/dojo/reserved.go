package dojo

import "slices"

// Reserved directory names under <base_path>/dojo. They hold shared calibration
// state (the corpus labels, the operator proposals and the published
// calibrations), so no scenario or case may take one of them as its name and
// `dojo list` never reports them as scenarios.
const (
	// DirCorpus holds the durable golden promotions and regression candidates.
	DirCorpus = "corpus"
	// DirMutations holds operator proposals pending approval.
	DirMutations = "mutations"
	// DirCalibrations holds the published strategist-calibration/v1 artifacts.
	DirCalibrations = "calibrations"
)

var reservedNames = []string{DirCorpus, DirMutations, DirCalibrations}

// ReservedNames returns the reserved directory names as a defensive copy.
func ReservedNames() []string {
	return slices.Clone(reservedNames)
}

// IsReservedName reports whether name is a reserved dojo directory name.
func IsReservedName(name string) bool {
	return slices.Contains(reservedNames, name)
}
