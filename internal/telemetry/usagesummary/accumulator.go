package usagesummary

type contribution struct {
	phase      string
	phaseRunID string
	in, out    *int64
	quality    string
}

type accumulator struct {
	in, out               int64
	haveIn, haveOut       bool
	missingIn, missingOut int
	qualities             map[string]int
	observations          int
}

type snapshotPoint struct{ in, out *int64 }

func (a *accumulator) add(c contribution) {
	if c.in == nil && c.out == nil {
		return
	}
	a.observations++
	if a.qualities == nil {
		a.qualities = map[string]int{}
	}
	a.qualities[c.quality]++
	if c.in != nil {
		a.in, a.haveIn = a.in+*c.in, true
	} else {
		a.missingIn++
	}
	if c.out != nil {
		a.out, a.haveOut = a.out+*c.out, true
	} else {
		a.missingOut++
	}
}

func (a *accumulator) bucket() Bucket {
	b := Bucket{Quality: QualityUnavailable, Observations: a.observations}
	if a.haveIn {
		v := a.in
		b.Tokens.Input = &v
	}
	if a.haveOut {
		v := a.out
		b.Tokens.Output = &v
	}
	if a.haveIn && a.haveOut && a.missingIn == 0 && a.missingOut == 0 {
		v := a.in + a.out
		b.Tokens.Total = &v
	}
	switch len(a.qualities) {
	case 0:
	case 1:
		for q := range a.qualities {
			b.Quality = q
		}
	default:
		b.Quality = QualityMixed
		b.QualityBreakdown = a.qualities
	}
	return b
}
