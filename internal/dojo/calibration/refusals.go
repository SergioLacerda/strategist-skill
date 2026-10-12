package calibration

// Refusals says why the golden parents were not approved at a threshold, run by
// run: the aggregate choice was not accept, the choice was accept but its
// probability did not exceed the threshold (an equal one does not), or the call
// gave no usable answer. Lowering the threshold can only help the second kind.
type Refusals struct {
	Threshold       float64 `json:"threshold"`
	Runs            int     `json:"runs"`
	Approved        int     `json:"approved"`
	ChoiceNotAccept int     `json:"choice_not_accept"`
	BelowThreshold  int     `json:"accept_at_or_below_threshold"`
	Invalid         int     `json:"no_usable_answer"`
}

func refusalsOf(stats []CaseStat, threshold float64) Refusals {
	r := Refusals{Threshold: threshold}
	for _, c := range positives(stats) {
		r.Invalid += c.Failed
		r.Runs += c.Failed
		for _, obs := range c.Observations {
			r.Runs++
			switch {
			case obs.Approved(threshold):
				r.Approved++
			case obs.Decision != "accept":
				r.ChoiceNotAccept++
			default:
				r.BelowThreshold++
			}
		}
	}
	return r
}
