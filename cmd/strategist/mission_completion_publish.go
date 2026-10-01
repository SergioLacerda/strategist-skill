package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
)

// completionFaultHook lets tests stop a completion at a named stage to prove
// crash recovery. It is nil in production.
var completionFaultHook func(stage string) error

const (
	faultBeforePublish = "before_publish"
	faultAfterPublish  = "after_publish"
)

func completionFault(stage string) error {
	if completionFaultHook == nil {
		return nil
	}
	return completionFaultHook(stage)
}

// publishDiscoveryArtifact runs the journaled publication under the target
// lease: normalize, commit digest and target (processing), publish, finalize
// (completed). A request left in processing is recovered idempotently: a
// matching artifact is finalized without normalizing again, an absent one is
// published from the same digest, and anything else fails closed.
func publishDiscoveryArtifact(ctx context.Context, store missionruntime.InvocationStore, input missionadapter.InvocationCompleteInput, record domain.MissionInvocationRecord) (string, error) {
	artifactPath, artifactAbsolute, err := discoveryArtifactPaths(input.Root, input.BasePath, record.Request.MissionID)
	if err != nil {
		return "", fmt.Errorf("normalize discovery completion: %w", err)
	}
	if finalized, err := resumeCommittedCompletion(store, record, artifactAbsolute); err != nil || finalized {
		return artifactPath, err
	}
	artifact, err := normalizeDiscoveryInvocation(ctx, input.Sink, record, artifactPath, input.Completion.Result)
	if err != nil {
		return "", err
	}
	if err := requireTargetFree(artifactAbsolute, record.Request.RequestID); err != nil {
		return "", err
	}
	if err := commitAndWrite(store, record.Request.RequestID, artifactPath, artifactAbsolute, artifact.Content); err != nil {
		return "", err
	}
	return artifactPath, nil
}

// commitAndWrite is the journaled write: commit the digest and target
// (processing), publish the bytes, then finalize (completed).
func commitAndWrite(store missionruntime.InvocationStore, requestID, artifactPath, artifactAbsolute string, content []byte) error {
	digest := contentDigest(content)
	if err := store.BeginProcessing(requestID, artifactPath, digest); err != nil {
		return fmt.Errorf("begin mission invocation: %w", err)
	}
	if err := completionFault(faultBeforePublish); err != nil {
		return err
	}
	if err := writeMissionArtifact(artifactAbsolute, content); err != nil {
		return err
	}
	if err := completionFault(faultAfterPublish); err != nil {
		return err
	}
	if err := store.Complete(requestID, digest); err != nil {
		return fmt.Errorf("complete mission invocation: %w", err)
	}
	return nil
}

// resumeCommittedCompletion recovers a processing request; a pending request has
// nothing to resume. It reports true once
// the published artifact is proven to be this request's and the journal is
// completed; false means the artifact is absent and publication must be retried.
func resumeCommittedCompletion(store missionruntime.InvocationStore, record domain.MissionInvocationRecord, path string) (bool, error) {
	if record.EffectiveState() != domain.InvocationStateProcessing {
		return false, nil
	}
	existing, err := os.ReadFile(path) //nolint:gosec // G304: path is built by discoveryArtifactPaths inside the workspace.
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect existing discovery artifact: %w", err)
	}
	provenance, err := provider.ParseDiscoveryProvenance(existing)
	if err != nil || provenance.RequestID != record.Request.RequestID || provenance.Status != "ranger_pending" || contentDigest(existing) != record.ArtifactDigest {
		return false, fmt.Errorf("invocation_artifact_conflict: %s does not match the artifact request %q committed; remove it to recover", filepath.Base(path), record.Request.RequestID)
	}
	if err := store.Complete(record.Request.RequestID, record.ArtifactDigest); err != nil {
		return false, fmt.Errorf("complete mission invocation: %w", err)
	}
	return true, nil
}

// requireTargetFree refuses to publish over any existing artifact. Ownership is
// read from parsed frontmatter, never body text; because a request commits to
// processing before it writes, a pre-existing artifact is always another
// request's, a promoted one, or one whose ownership cannot be established.
func requireTargetFree(path, requestID string) error {
	existing, err := os.ReadFile(path) //nolint:gosec // G304: path is built by discoveryArtifactPaths inside the workspace.
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect existing discovery artifact: %w", err)
	}
	provenance, err := provider.ParseDiscoveryProvenance(existing)
	switch {
	case err != nil:
		return fmt.Errorf("invocation_artifact_exists: %s has unreadable frontmatter and will not be overwritten: %w", filepath.Base(path), err)
	case provenance.Status != "ranger_pending":
		return fmt.Errorf("invocation_artifact_exists: %s is not a pending Ranger artifact and will not be overwritten", filepath.Base(path))
	case provenance.RequestID == "":
		return fmt.Errorf("invocation_artifact_exists: %s is pending but carries no request provenance; remove it to issue a new request", filepath.Base(path))
	default:
		return fmt.Errorf("invocation_artifact_exists: %s is owned by request %q, not %q", filepath.Base(path), provenance.RequestID, requestID)
	}
}

func contentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
