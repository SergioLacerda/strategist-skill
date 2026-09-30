package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/spf13/cobra"
)

func completeMissionInvocation(ctx context.Context, input missionadapter.InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
	store := missionruntime.NewInvocationStore(input.Root)
	record, err := loadCompletableInvocation(store, input)
	if err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	artifactPath, artifactAbsolute, err := discoveryArtifactPaths(input.Root, input.BasePath, record.Request.MissionID)
	if err != nil {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("normalize discovery completion: %w", err)
	}
	artifact, err := normalizeDiscoveryInvocation(ctx, record, artifactPath, input.Completion.Result)
	if err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	if err := writeMissionArtifact(artifactAbsolute, artifact.Content); err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	if err := store.Consume(input.RequestID); err != nil {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("consume mission invocation: %w", err)
	}
	return domain.MissionInvocationOutcome{RequestID: input.RequestID, MissionID: record.Request.MissionID, Status: "normalized", ArtifactPath: artifactPath, BindingDigest: record.Request.BindingDigest, SourceDigest: record.Request.SourceDigest}, nil
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
	now := time.Now().UTC()
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
				SourceDigest: record.Request.SourceDigest, IssuedAt: now,
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
	tmp, err := os.CreateTemp(filepath.Dir(path), ".analysis-*.tmp")
	if err != nil {
		return fmt.Errorf("write normalized artifact: create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() //nolint:errcheck // best-effort cleanup of an uncommitted temporary artifact.
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close() //nolint:errcheck // best-effort cleanup before returning the chmod error.
		return fmt.Errorf("write normalized artifact: protect temporary file: %w", err)
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close() //nolint:errcheck // best-effort cleanup before returning the write error.
		return fmt.Errorf("write normalized artifact: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write normalized artifact: close temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("write normalized artifact: persist: %w", err)
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
