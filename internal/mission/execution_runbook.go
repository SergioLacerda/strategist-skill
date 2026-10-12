package mission

import (
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// runbookGroup is the main markdown plus its typed `.runbook.yaml` sidecar (the
// format the selector and ParseSidecar define). The sidecar identity is derived
// from the main file so the two agree without a manual rename: runbook_id is
// the markdown stem and source_doc is its project-relative path.
func runbookGroup(key domain.ResolverKey, sideQuestID, mainPath string) domain.ExecutionGroup {
	stem := strings.TrimSuffix(filepath.Base(mainPath), filepath.Ext(mainPath))
	sidecar := strings.TrimSuffix(mainPath, filepath.Ext(mainPath)) + ".runbook.yaml"
	groupID := "runbook-" + sideQuestID
	return domain.ExecutionGroup{
		ID: groupID, Resolver: key, SideQuestID: sideQuestID, Kind: "runbook", Origin: domain.ExecutionTargetOriginAcceptedSideQuest,
		CompletionPolicy: domain.ExecutionCompletionAllRequired,
		Artifacts: []domain.ExecutionArtifact{
			{ID: groupID + "-main", Role: domain.ExecutionArtifactRolePrimary, Path: mainPath, Kind: "runbook"},
			{ID: groupID + "-sidecar", Role: domain.ExecutionArtifactRoleSidecar, Path: sidecar, Kind: domain.RunbookSidecarKind,
				Metadata: map[string]string{"runbook_id": stem, "source_doc": mainPath}},
		},
	}
}
