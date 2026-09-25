package metrics

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// recordSingleClaim records the one `claim:` of a document.
func recordSingleClaim(cmd *cobra.Command, producer telemetry.ConfidenceProducerAdapter, input confidenceClaimFile) error {
	record, err := producer.RecordClaim(input.Claim, input.Evidence)
	if err != nil {
		return fmt.Errorf("metrics record: %w", err)
	}
	if err := printRecorded(cmd, record); err != nil {
		return err
	}
	// The rejected observation is persisted for the confidence metrics, but the
	// command still fails: an exit status of 0 let rejections go unnoticed.
	if record.CoverageStatus == telemetry.ConfidenceCoverageRejected {
		return fmt.Errorf("metrics record: claim %s rejected (recorded as rejected): %s", record.ClaimID, record.Violation)
	}
	return nil
}

func printRecorded(cmd *cobra.Command, record telemetry.ConfidenceRecord) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "coverage_status: %s\nclaim: %s\nviolation: %s\n", record.CoverageStatus, record.ClaimID, record.Violation); err != nil {
		return fmt.Errorf("metrics record: write output: %w", err)
	}
	return nil
}

// recordClaimBatch records every declared claim of a `claims:` document, so the
// confidence history holds the whole declared set instead of one claim per
// command. The batch is validated as a whole before anything is written; each
// claim then keeps the per-claim rule: an invalid one is persisted as a
// rejected observation and the command fails naming it.
func recordClaimBatch(cmd *cobra.Command, producer telemetry.ConfidenceProducerAdapter, input confidenceClaimFile) error {
	if err := validateBatch(producer.Agent, input.Claims); err != nil {
		return err
	}
	rejected, err := recordEach(cmd, producer, input)
	if err != nil {
		return err
	}
	return summarizeBatch(cmd, len(input.Claims), rejected)
}

// recordEach records the claims in order and returns the ids that were
// persisted as rejected.
func recordEach(cmd *cobra.Command, producer telemetry.ConfidenceProducerAdapter, input confidenceClaimFile) ([]string, error) {
	var rejected []string
	for _, claim := range input.Claims {
		record, err := producer.RecordClaim(claim, input.Evidence)
		if err != nil {
			return nil, fmt.Errorf("metrics record: %w", err)
		}
		if err := printRecorded(cmd, record); err != nil {
			return nil, err
		}
		if record.CoverageStatus == telemetry.ConfidenceCoverageRejected {
			rejected = append(rejected, record.ClaimID)
		}
	}
	return rejected, nil
}

func summarizeBatch(cmd *cobra.Command, total int, rejected []string) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "recorded: %d (reported %d, rejected %d)\n", total, total-len(rejected), len(rejected)); err != nil {
		return fmt.Errorf("metrics record: write output: %w", err)
	}
	if len(rejected) > 0 {
		return fmt.Errorf("metrics record: %d of %d claims rejected (recorded as rejected): %s", len(rejected), total, strings.Join(rejected, ", "))
	}
	return nil
}

// validateBatch fails closed before any write: a claim without an id cannot be
// recorded, a claim declared by another agent would be attributed to the
// producing agent, and a repeated id would be counted as a duplicate.
func validateBatch(agent string, claims []domain.ConfidenceClaim) error {
	seen := make(map[string]struct{}, len(claims))
	for i, claim := range claims {
		if err := validateBatchClaim(agent, i, claim, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateBatchClaim(agent string, index int, claim domain.ConfidenceClaim, seen map[string]struct{}) error {
	if claim.ID == "" {
		return fmt.Errorf("metrics record: claim #%d has no id; nothing recorded", index+1)
	}
	if claim.Agent != "" && claim.Agent != agent {
		return fmt.Errorf("metrics record: claim %s is declared by agent %q but is being recorded as %q; record each agent's claims with its own --agent", claim.ID, claim.Agent, agent)
	}
	if _, dup := seen[claim.ID]; dup {
		return fmt.Errorf("metrics record: duplicate claim id %s in the claims list; nothing recorded", claim.ID)
	}
	seen[claim.ID] = struct{}{}
	return nil
}
