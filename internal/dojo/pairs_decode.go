package dojo

import (
	"encoding/json"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
)

func decodePair(raw []byte) (calibration.PairObservation, bool) {
	var o calibration.PairObservation
	if json.Unmarshal(raw, &o) != nil || o.Pair == "" {
		return calibration.PairObservation{}, false
	}
	return o, true
}
