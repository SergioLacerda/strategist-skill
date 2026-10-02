package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/filelock"
)

// BeginProcessing durably commits a completion to publish artifactDigest at
// targetPath (pending -> processing). Re-entering processing is allowed only
// with the same target and digest, which makes crash recovery idempotent.
func (s InvocationStore) BeginProcessing(requestID, targetPath, artifactDigest string) error {
	record, err := s.Get(requestID)
	if err != nil {
		return err
	}
	if targetPath == "" || artifactDigest == "" {
		return fmt.Errorf("mission invocation: target path and artifact digest are required to start processing")
	}
	if err := requireTransition(record, domain.InvocationStateProcessing); err != nil {
		return err
	}
	if err := requireSameCommitment(record, targetPath, artifactDigest); err != nil {
		return err
	}
	now := s.now()
	record.State, record.TargetPath, record.ArtifactDigest = domain.InvocationStateProcessing, targetPath, artifactDigest
	if record.ProcessingAt == nil {
		record.ProcessingAt = &now
	}
	return s.write(requestID, record)
}

// Complete finalizes a processing request (processing -> completed). It
// compacts the record to identity, digests, state and timestamps: the payload
// and request context are not needed for replay rejection or recovery.
func (s InvocationStore) Complete(requestID, artifactDigest string) error {
	record, err := s.Get(requestID)
	if err != nil {
		return err
	}
	if err := requireTransition(record, domain.InvocationStateCompleted); err != nil {
		return err
	}
	if record.ArtifactDigest != artifactDigest {
		return fmt.Errorf("invocation_digest_mismatch: request %q committed a different artifact digest", requestID)
	}
	now := s.now()
	record.State, record.Consumed, record.ConsumedAt, record.CompletedAt = domain.InvocationStateCompleted, true, &now, &now
	record.Request.Payload, record.Request.Input = "", nil
	return s.write(requestID, record)
}

// requireSameCommitment keeps a recovery on the artifact the first attempt
// committed to; a pending record has no commitment yet.
func requireSameCommitment(record domain.MissionInvocationRecord, targetPath, artifactDigest string) error {
	if record.EffectiveState() != domain.InvocationStateProcessing {
		return nil
	}
	if record.TargetPath != targetPath || record.ArtifactDigest != artifactDigest {
		return fmt.Errorf("invocation_digest_mismatch: request %q is already committed to a different artifact", record.Request.RequestID)
	}
	return nil
}

func requireTransition(record domain.MissionInvocationRecord, next domain.MissionInvocationState) error {
	if !record.EffectiveState().CanTransitionTo(next) {
		return fmt.Errorf("invocation_state_invalid: request %q cannot move from %s to %s", record.Request.RequestID, record.EffectiveState(), next)
	}
	return nil
}

func (s InvocationStore) write(requestID string, record domain.MissionInvocationRecord) error {
	path, err := s.path(requestID)
	if err != nil {
		return err
	}
	return atomicWriteJSON(path, record)
}

// CommitExecutionAdapter records the child adapter Strategist is about to
// launch for a pending request, before the child runs. Only Strategist-owned
// dispatch calls it. The commitment is one-way: a record that was not issued
// as current_host_adapter (including a pre-field record) is never upgraded,
// and a committed child mode cannot change.
func (s InvocationStore) CommitExecutionAdapter(requestID string, adapter domain.MissionExecutionAdapter, policyID string) error {
	path, err := s.path(requestID)
	if err != nil {
		return err
	}
	if err := filelock.WithLock(path, func() error {
		return s.commitExecutionAdapter(requestID, adapter, policyID)
	}); err != nil {
		return fmt.Errorf("commit invocation adapter: %w", err)
	}
	return nil
}

func (s InvocationStore) commitExecutionAdapter(requestID string, adapter domain.MissionExecutionAdapter, policyID string) error {
	record, err := s.Get(requestID)
	if err != nil {
		return err
	}
	if !adapter.IsChild() || policyID == "" {
		return fmt.Errorf("invocation_adapter_unknown: %q is not a committable child adapter with a policy identity", adapter)
	}
	if record.EffectiveState() != domain.InvocationStatePending {
		return fmt.Errorf("invocation_state_invalid: request %q can only commit an adapter while pending", requestID)
	}
	if record.ExecutionAdapter == adapter && record.ChildPolicyID == policyID {
		return nil
	}
	if record.ExecutionAdapter != domain.ExecutionAdapterCurrentHost {
		return fmt.Errorf("invocation_adapter_mismatch: request %q was issued as %s and cannot become %s", requestID, record.EffectiveAdapter(), adapter)
	}
	record.ExecutionAdapter, record.ChildPolicyID = adapter, policyID
	return s.write(requestID, record)
}
