package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/spf13/cobra"
)

func completeMissionInvocation(ctx context.Context, input missionadapter.InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
	store := missionruntime.NewInvocationStore(input.Root)
	release, err := store.Claim(input.RequestID)
	if err != nil {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("claim mission invocation: %w", err)
	}
	defer release()
	record, err := loadCompletableInvocation(store, input)
	if err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	if err := verifyInvocationStillCurrent(input.Root, record); err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	artifactPath, err := persistDiscoveryArtifact(ctx, input, record)
	if err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	if err := store.Consume(input.RequestID); err != nil {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("consume mission invocation: %w", err)
	}
	return domain.MissionInvocationOutcome{RequestID: input.RequestID, MissionID: record.Request.MissionID, Status: "normalized", ArtifactPath: artifactPath, BindingDigest: record.Request.BindingDigest, SourceDigest: record.Request.SourceDigest}, nil
}

// persistDiscoveryArtifact normalizes the raw completion and writes the pending
// Ranger artifact, returning its workspace-relative path.
func persistDiscoveryArtifact(ctx context.Context, input missionadapter.InvocationCompleteInput, record domain.MissionInvocationRecord) (string, error) {
	artifactPath, artifactAbsolute, err := discoveryArtifactPaths(input.Root, input.BasePath, record.Request.MissionID)
	if err != nil {
		return "", fmt.Errorf("normalize discovery completion: %w", err)
	}
	artifact, err := normalizeDiscoveryInvocation(ctx, record, artifactPath, input.Completion.Result)
	if err != nil {
		return "", err
	}
	if err := refuseArtifactOverwrite(artifactAbsolute); err != nil {
		return "", err
	}
	if err := writeMissionArtifact(artifactAbsolute, artifact.Content); err != nil {
		return "", err
	}
	return artifactPath, nil
}

// verifyInvocationStillCurrent rejects a request whose mission left discovery
// or whose Ranked binding changed after the request was issued.
func verifyInvocationStillCurrent(root string, record domain.MissionInvocationRecord) error {
	phase, found := missionruntime.ReadMissionPhase(root, record.Request.MissionID)
	if !found || phase != domain.PhaseDiscovery {
		return fmt.Errorf("invocation_phase_mismatch: mission %q is not in %s (found=%t, phase=%q)", record.Request.MissionID, domain.PhaseDiscovery, found, phase)
	}
	binding, _, _, _, err := resolveMissionInvocationWeapon(missionadapter.InvocationBuildInput{
		Root: root, MissionID: record.Request.MissionID, Role: record.Request.Role, Slot: record.Request.Slot,
	})
	if err != nil {
		return err
	}
	if binding.BindingDigest != record.Request.BindingDigest {
		return fmt.Errorf("invocation_binding_mismatch: the active binding changed since the request was issued")
	}
	return nil
}

// refuseArtifactOverwrite lets a completion replace only a pending Ranger
// artifact; a promoted, approved or claimed artifact is never overwritten.
func refuseArtifactOverwrite(path string) error {
	existing, err := os.ReadFile(path) //nolint:gosec // G304: path is built by discoveryArtifactPaths inside the workspace.
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect existing discovery artifact: %w", err)
	}
	if !bytes.Contains(existing, []byte("mission_status: ranger_pending")) {
		return fmt.Errorf("invocation_artifact_exists: %s is not a pending Ranger artifact and will not be overwritten", filepath.Base(path))
	}
	return nil
}

func loadCompletableInvocation(store missionruntime.InvocationStore, input missionadapter.InvocationCompleteInput) (domain.MissionInvocationRecord, error) {
	record, err := store.Get(input.RequestID)
	if err != nil {
		return domain.MissionInvocationRecord{}, fmt.Errorf("load mission invocation: %w", err)
	}
	if err := input.Completion.Validate(); err != nil {
		return domain.MissionInvocationRecord{}, fmt.Errorf("validate mission completion: %w", err)
	}
	if input.Completion.RequestID != record.Request.RequestID {
		return domain.MissionInvocationRecord{}, fmt.Errorf("invocation_binding_mismatch: completion request_id does not match pending request")
	}
	if record.Request.Role != "ranger" || record.Request.Slot != string(domain.SlotDiscovery) {
		return domain.MissionInvocationRecord{}, fmt.Errorf("role_invocation_failed: completion normalization is not registered for %s/%s", record.Request.Role, record.Request.Slot)
	}
	return record, nil
}

func normalizeDiscoveryInvocation(ctx context.Context, record domain.MissionInvocationRecord, artifactPath, result string) (provider.NormalizedDiscoveryArtifact, error) {
	request := provider.DiscoveryWeaponRequest{
		MissionID: record.Request.MissionID, Role: record.Request.Role, Slot: record.Request.Slot,
		ProviderID: record.Request.Weapon.ID, RuntimeKind: domain.RankedRuntimeEmbedded,
		BindingDigest: record.Request.BindingDigest, SourceDigest: record.Request.SourceDigest,
		Entrypoint: record.Request.Entrypoint, ArtifactPath: artifactPath,
	}
	artifact, err := provider.InvokeAndNormalizeDiscovery(ctx, request, func(context.Context, provider.DiscoveryWeaponRequest) (provider.DiscoveryWeaponResponse, error) {
		return provider.DiscoveryWeaponResponse{
			ProviderID: record.Request.Weapon.ID, InvocationEvidence: "embedded_prompt_bridge",
			Artifact: []byte(result), EmbeddedInvocationReceipt: connectors.EmbeddedInvocationReceipt{
				SchemaVersion: connectors.EmbeddedInvocationReceiptSchemaVersion, MissionID: record.Request.MissionID,
				Role: record.Request.Role, Slot: record.Request.Slot, WeaponID: record.Request.Weapon.ID,
				Entrypoint: record.Request.Entrypoint, BindingDigest: record.Request.BindingDigest,
				SourceDigest: record.Request.SourceDigest, RequestID: record.Request.RequestID, IssuedAt: record.CreatedAt,
			},
		}, nil
	})
	if err != nil {
		return provider.NormalizedDiscoveryArtifact{}, fmt.Errorf("normalize discovery completion: %w", err)
	}
	return artifact, nil
}

func discoveryArtifactPaths(root, basePath, missionID string) (string, string, error) {
	workspace := filepath.Dir(root)
	absolute := filepath.Join(basePath, "pending", missionID+"-analysis.md")
	relative, err := filepath.Rel(workspace, absolute)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || len(relative) < 3 || relative[:3] == "../" {
		return "", "", fmt.Errorf("role_invocation_failed: discovery artifact path escapes workspace")
	}
	return filepath.ToSlash(relative), absolute, nil
}

func writeMissionArtifact(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write normalized artifact: create directory: %w", err)
	}
	if err := missionruntime.WriteFileAtomic(path, content, 0o644); err != nil {
		return fmt.Errorf("write normalized artifact: %w", err)
	}
	return nil
}

func readMissionCompletion(cmd *cobra.Command) (domain.MissionInvocationCompletion, error) {
	var completion domain.MissionInvocationCompletion
	decoder := json.NewDecoder(cmd.InOrStdin())
	if err := decoder.Decode(&completion); err != nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: more than one object was supplied")
	} else if err != io.EOF {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: trailing data: %w", err)
	}
	return completion, nil
}
