package gatecriteria

import "strings"

const stageVolumeCriterion = "token_usage.by_stage"

// stageTokensRow is the criteria row for the per-stage volumes. It is
// evaluated only when a volume can be stated; otherwise it says why not.
func stageTokensRow(st StageTokens) Row {
	switch st.Availability {
	case AvailabilityAvailable:
		return evaluated(stageVolumeCriterion, stageRowValue(st), usageSource, stageRowDetail(st))
	case AvailabilityNotApplicable:
		return notEvaluated(stageVolumeCriterion, usageSource, "nenhum uso reportado ainda")
	}
	return unknown(stageVolumeCriterion, usageSource, "o ledger de uso não pôde ser lido ou não sustenta um volume")
}

func stageRowValue(st StageTokens) string {
	parts := make([]string, 0, len(st.Stages)+1)
	for _, s := range st.Stages {
		parts = append(parts, s.Stage+" "+figure(s.Tokens, s.LowerBound))
	}
	parts = append(parts, "total "+figure(st.Total.Tokens, boundOf(st.Total.Tokens)))
	return strings.Join(parts, "; ")
}

func stageRowDetail(st StageTokens) string {
	detail := "volume por estágio, qualidade " + st.Total.Quality + "; sem custo ou preço"
	for _, s := range st.Stages {
		if s.Pending {
			return detail + "; " + s.Stage + " só existe depois do gate"
		}
	}
	return detail
}
