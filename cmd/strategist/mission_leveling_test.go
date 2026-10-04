package main

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/feats/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/require"
)

// TestResolveMissionLevelingUnresolvableScoutIsNeverPersisted covers F-L4
// (ADR-0057/design.md task 4.4): resolveMissionLeveling's !found branch has
// no host or provider data to resolve from (mission start runs before any
// role's own on_start hook), so it can only ever produce an Unknown() level.
// That is now returned as ObservationUnavailable without appending a ledger
// row or emitting role_level_resolved for a resolution that did not happen —
// a row with empty model/effort/level_source carried no information under
// the old behavior, which this replaces.
func TestResolveMissionLevelingUnresolvableScoutIsNeverPersisted(t *testing.T) {
	root := t.TempDir()
	first, err := resolveMissionLeveling(root, "mission-level", "scout", "run-1")
	require.NoError(t, err)
	require.Equal(t, initiative.ObservationUnavailable, first.State)
	require.NotEmpty(t, first.EventID)

	path := filepath.Join(root, "memory", roleLevelLedger)
	records, err := leveling.ReadRecords(path)
	require.NoError(t, err)
	require.Empty(t, records, "an unresolvable scout level must not be appended to the LEVELING ledger")

	// A repeated call still reports the same advisory state — never found,
	// never persisted, never crashing — even though nothing is cached for
	// this always-empty case.
	second, err := resolveMissionLeveling(root, "mission-level", "scout", "run-1")
	require.NoError(t, err)
	require.Equal(t, initiative.ObservationUnavailable, second.State)
	records, err = leveling.ReadRecords(path)
	require.NoError(t, err)
	require.Empty(t, records)
}

// TestResolveMissionLevelingReusesAnActuallyResolvedRecord confirms the
// ledger-reuse path is untouched for the case that matters: a role that DID
// resolve to a real level (for example after its own on_start hook ran and
// recorded one) is read back and reused, not re-resolved.
func TestResolveMissionLevelingReusesAnActuallyResolvedRecord(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "memory", roleLevelLedger)
	require.NoError(t, leveling.AppendRecord(path, leveling.Record{
		MissionID: "mission-level", Run: "run-1",
		Level: leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: leveling.SourceHost},
	}))

	resolution, err := resolveMissionLeveling(root, "mission-level", "ranger", "run-1")
	require.NoError(t, err)
	require.Equal(t, initiative.ObservationKnown, resolution.State)
	require.Equal(t, "Sonnet", resolution.Model)

	records, err := leveling.ReadRecords(path)
	require.NoError(t, err)
	require.Len(t, records, 1, "reading an existing record must not append a second one")
}

// TestStartInitiativeConsultationUsesThePersistedLevelingSnapshot covers the
// INITIATIVE integration for a role that DID resolve — the case
// resolveMissionLeveling's ledger-reuse path exists for. The always-empty
// mission-start/scout case (F-L4) is covered separately above and correctly
// records nothing.
func TestStartInitiativeConsultationUsesThePersistedLevelingSnapshot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "memory", roleLevelLedger)
	require.NoError(t, leveling.AppendRecord(path, leveling.Record{
		MissionID: "mission-consult", Run: "mission-consult",
		Level: leveling.Level{Role: "scout", Model: "Sonnet", Effort: "high", Source: leveling.SourceHost},
	}))

	require.NoError(t, startInitiativeConsultation(root, "mission-consult"))

	levelRecords, err := leveling.ReadRecords(path)
	require.NoError(t, err)
	require.Len(t, levelRecords, 1, "startInitiativeConsultation must not append a second LEVELING row when one already resolved")
	initiativeRecords, err := initiative.ReadRecords(initiative.LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, initiativeRecords, 1)
	require.NotNil(t, initiativeRecords[0].Advice.Leveling)
	require.Equal(t, levelRecords[0].MissionID, initiativeRecords[0].Advice.MissionID)
	require.Equal(t, levelRecords[0].Role, initiativeRecords[0].Advice.Leveling.Role)
	require.Equal(t, "leveling-mission-consult-mission-consult-"+levelRecords[0].Timestamp, initiativeRecords[0].Advice.Leveling.EventID)
}

// TestStartInitiativeConsultationWithNoPriorLevelStillSucceeds confirms
// INITIATIVE consultation itself does not fail or block mission start when
// the role's level has genuinely not resolved yet (F-L4's own scenario) — it
// records an ObservationUnavailable advice, not an error, and the LEVELING
// ledger stays empty.
func TestStartInitiativeConsultationWithNoPriorLevelStillSucceeds(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, startInitiativeConsultation(root, "mission-empty"))

	levelRecords, err := leveling.ReadRecords(filepath.Join(root, "memory", roleLevelLedger))
	require.NoError(t, err)
	require.Empty(t, levelRecords)

	initiativeRecords, err := initiative.ReadRecords(initiative.LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, initiativeRecords, 1)
	require.Equal(t, initiative.ObservationUnavailable, initiativeRecords[0].Advice.Leveling.State)
}
