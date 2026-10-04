package application

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestValidateInvocationCompletionRejectsInvalidRecords(t *testing.T) {
	base := domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{RequestID: "req", Role: "ranger", Slot: string(domain.SlotDiscovery)}}
	cases := []struct {
		name       string
		record     domain.MissionInvocationRecord
		completion domain.MissionInvocationCompletion
		want       string
	}{
		{"empty completion", base, domain.MissionInvocationCompletion{}, "validate mission completion"},
		{"wrong request", base, domain.MissionInvocationCompletion{RequestID: "other", Result: "ok"}, "invocation_binding_mismatch"},
		{"unregistered role", domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{RequestID: "req", Role: "archivist", Slot: "refinement"}}, domain.MissionInvocationCompletion{RequestID: "req", Result: "ok"}, "not registered"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			require.ErrorContains(t, ValidateInvocationCompletion(test.record, test.completion), test.want)
		})
	}
}

func TestVerifyExecutionAdapterRejectsUnknownAndMismatchedAdapters(t *testing.T) {
	base := domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{RequestID: "req"}, ExecutionAdapter: domain.ExecutionAdapterCurrentHost}
	require.ErrorContains(t, VerifyExecutionAdapter(base, ""), "adapter_missing")
	require.ErrorContains(t, VerifyExecutionAdapter(base, "invalid"), "adapter_unknown")

	child := base
	child.ExecutionAdapter = domain.ExecutionAdapterCodexChild
	require.ErrorContains(t, VerifyExecutionAdapter(child, domain.ExecutionAdapterClaudeChild), "adapter_mismatch")
	require.ErrorContains(t, VerifyExecutionAdapter(child, domain.ExecutionAdapterCodexChild), "no policy identity")
	child.ChildPolicyID = "policy/v1"
	require.NoError(t, VerifyExecutionAdapter(child, domain.ExecutionAdapterCodexChild))
}
