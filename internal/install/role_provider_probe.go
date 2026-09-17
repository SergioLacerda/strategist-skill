package install

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
)

func connectorProbe(binding domain.SlotBinding, instance domain.InstalledInstance) lifecycle.ProbeOutcome {
	entrypoint := "refine"
	if binding.Slot == string(domain.SlotDiscovery) {
		entrypoint = "discover"
	}
	result := connectors.LocalPathConnector{
		ConnectorID:         instance.ConnectorID,
		ConnectorAPIVersion: "strategist-connector-api/1",
	}.Probe(context.Background(), instance, entrypoint)
	return lifecycle.ProbeOutcome{Status: result.Status, ReasonCode: result.ReasonCode, Detail: result.Detail}
}
