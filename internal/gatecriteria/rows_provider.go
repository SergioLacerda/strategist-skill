package gatecriteria

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
)

const providerSource = "missions/handoff + memory/integration-calls.jsonl"

// providerRow reports the provider reading of one boundary: the delegation
// recorded on the handoff outcome when there is one, otherwise the call ledger.
func providerRow(in Input, criterion, transition, beforeGate string) Row {
	if delegation := outcomeDelegation(in, transition); delegation != nil {
		return evaluated(criterion, readingText(delegation.Provider, delegation.Confidence, delegation.Threshold, calibrationText(in, delegation.CalibrationID, "")), providerSource, "delegação registrada no outcome")
	}
	ledger := handoffconsumer.Ledger{Path: in.memory("integration-calls.jsonl")}
	record, found, err := ledger.LatestRecord(in.MissionID, transition)
	switch {
	case err != nil:
		return unknown(criterion, providerSource, "o ledger de chamadas não pôde ser lido")
	case !found:
		return notEvaluated(criterion, providerSource, absentDetail(in, beforeGate))
	case !record.Answered:
		return notEvaluated(criterion, providerSource, "provedor não respondeu: "+orDefault(record.Reason, "motivo não informado"))
	case transition == handoffconsumer.TransitionArchivistToGate && !currentRevision(in, record.Subject):
		return unknown(criterion, providerSource, "leitura de outra revisão do pacote")
	}
	note := calibrationText(in, appliedRecordID(record), record.CalibrationReason)
	return evaluated(criterion, readingText(record.Provider, record.Confidence, record.Threshold, note), providerSource, "leitura registrada no ledger de chamadas")
}

func outcomeDelegation(in Input, transition string) *handoff.Delegation {
	if transition == handoffconsumer.TransitionArchivistToGate {
		return nil
	}
	outcome, err := handoff.NewOutcomeStore(in.Root).LatestFor(in.MissionID, transition)
	if err != nil {
		return nil
	}
	return outcome.Delegation
}

// currentRevision reports whether a recorded subject is the package as it is now.
func currentRevision(in Input, subject string) bool {
	digest, err := handoff.PackageDigest(filepath.Join(in.BasePath, "refined", in.MissionID))
	return err == nil && digest == subject
}

func absentDetail(in Input, beforeGate string) string {
	file, found, err := config.Load(filepath.Join(in.Root, config.FileName))
	provider, declared := file.Providers["jev"]
	if err != nil || !found || !declared || !provider.Enabled {
		return "integração não habilitada neste workspace"
	}
	return beforeGate
}

// readingText states the provider confidence against the local threshold. The
// rule is the consumer's, applied here only to words: strictly above approves.
func readingText(provider string, confidence, threshold float64, calibration string) string {
	name := strings.ToUpper(orDefault(provider, "provedor"))
	verdict := "abaixo do limiar, caminho main"
	if confidence > threshold {
		verdict = "aprovado"
	}
	return fmt.Sprintf("%s confiança: %d%% (limiar %d%%) — %s, %s", name, percent(confidence), percent(threshold), verdict, calibration)
}

func percent(value float64) int { return int(math.Round(value * 100)) }
