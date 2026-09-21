package check

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimeenv"
)

func hostRankedRuntimeReadiness(runtimeRoot, provider, pin string) domain.ReadinessCheck {
	health := runRankedRuntimeHealthcheck(runtimeRoot, provider)
	if !health.Ready() {
		return health
	}
	if skew := hostVersionSkewCheck(runtimeRoot, provider, pin); skew != nil {
		return *skew
	}
	return health
}

// hostVersionSkewCheck compares a host executable's --version with the
// contract pin. It returns a Ready check carrying the skew reason (advisory,
// never a block: the private runtime is the certified path) or nil when the
// contract is unpinned or the versions match.
func hostVersionSkewCheck(runtimeRoot, provider, pin string) *domain.ReadinessCheck {
	if pin == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), rankedHealthcheckTimeout())
	defer cancel()
	observed := ""
	if cmd, err := runtimeenv.Command(ctx, runtimeRoot, "openspec", "--version"); err == nil {
		if out, runErr := cmd.Output(); runErr == nil {
			observed = domain.ParseReportedVersion(out)
		}
	}
	if !domain.VersionSkew(pin, observed) {
		return nil
	}
	return &domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: domain.ReasonRankedRuntimeVersionSkew, Detail: domain.RankedRuntimeVersionSkewMessage(provider, pin, observed)}
}
