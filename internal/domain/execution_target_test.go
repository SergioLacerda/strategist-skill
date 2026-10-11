package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutionTargetRoundTrip(t *testing.T) {
	want := ExecutionTarget{
		Path:        "archived/m-1-adr.md",
		Kind:        ExecutionTargetKindADR,
		Origin:      ExecutionTargetOriginAcceptedSideQuest,
		SideQuestID: "OA-ADR-m-1",
	}

	raw, err := json.Marshal(want)
	require.NoError(t, err)
	var got ExecutionTarget
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, want, got)
}

func TestExecutionTargetConstants(t *testing.T) {
	require.NotEmpty(t, ExecutionTargetKindDocumentation)
	require.NotEmpty(t, ExecutionTargetKindADR)
	require.NotEmpty(t, ExecutionTargetOriginRefinedPackage)
	require.NotEmpty(t, ExecutionTargetOriginAcceptedSideQuest)
}

func TestExecutionTargetValidateRejectsUnsafePaths(t *testing.T) {
	valid := ExecutionTarget{Path: "docs/adr/0042.md", Kind: ExecutionTargetKindADR, Origin: ExecutionTargetOriginAcceptedSideQuest, SideQuestID: "OA-ADR-m1"}
	require.NoError(t, valid.Validate())
	for _, path := range []string{"", "/tmp/adr.md", "../adr.md", `C:\\adr.md`} {
		invalid := valid
		invalid.Path = path
		require.Error(t, invalid.Validate(), path)
	}
	missingSideQuest := valid
	missingSideQuest.SideQuestID = ""
	require.Error(t, missingSideQuest.Validate())
}
