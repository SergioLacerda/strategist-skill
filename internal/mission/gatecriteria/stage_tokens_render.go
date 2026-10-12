package gatecriteria

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry/usagesummary"
)

const (
	noFigure         = "—"
	stageNameWidth   = 10
	stageHeader      = "tokens estimados:"
	noUsageText      = "nenhum uso reportado"
	unattributedText = "não atribuído"
)

// RenderTokenStages renders the per-stage volume block shown at the gate:
//
//	tokens estimados:
//	discovery   (111) -> refining    (222) -> delivering  (—)
//	total       (333)  reported_unverified
//
// An unobserved stage is a dash and a partially observed one is a lower bound
// (≥n); neither is ever zero. The header keeps the operator's wording: the
// quality of the numbers is on the total line.
func RenderTokenStages(st StageTokens) string {
	var b strings.Builder
	b.WriteString(stageHeader + "\n")
	parts := make([]string, 0, len(st.Stages))
	for _, s := range st.Stages {
		parts = append(parts, labelled(s.Stage, figure(s.Tokens, s.LowerBound)))
	}
	b.WriteString(strings.Join(parts, " -> ") + "\n")
	if st.Unattributed.Observations > 0 {
		fmt.Fprintf(&b, "%s (%s)\n", unattributedText, figure(st.Unattributed.Tokens, boundOf(st.Unattributed.Tokens)))
	}
	b.WriteString(totalLine(st) + "\n")
	return b.String()
}

func totalLine(st StageTokens) string {
	line := labelled("total", figure(st.Total.Tokens, boundOf(st.Total.Tokens)))
	if st.Total.Observations == 0 {
		return line + "  " + noUsageText
	}
	return line + "  " + st.Total.Quality
}

func labelled(name, fig string) string {
	return fmt.Sprintf("%-*s  (%s)", stageNameWidth, name, fig)
}

func boundOf(t usagesummary.Tokens) *int64 { return lowerBound(t) }

// figure renders a volume: the total, a lower bound or a dash.
func figure(t usagesummary.Tokens, bound *int64) string {
	switch {
	case t.Total != nil:
		return strconv.FormatInt(*t.Total, 10)
	case bound != nil:
		return "≥" + strconv.FormatInt(*bound, 10)
	}
	return noFigure
}
