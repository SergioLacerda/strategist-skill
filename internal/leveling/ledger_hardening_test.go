package leveling

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLedgerWritesAndMigratesSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-levels.jsonl")
	require.NoError(t, AppendRecord(path, Record{MissionID: "m", Level: Level{Role: "ranger", Model: "model", Effort: "high"}}))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(content), `"schema_version":"1"`)

	legacy := []byte(`{"mission_id":"legacy","role":"ranger","model":"model","effort":"high"}` + "\n")
	require.NoError(t, os.WriteFile(path, legacy, 0o600))
	record, found, err := LatestRecord(path, "legacy", "ranger")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, LedgerSchemaVersion, record.SchemaVersion)
}

func TestLedgerClassifiesOversizedRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-levels.jsonl")
	require.NoError(t, os.WriteFile(path, append(bytes.Repeat([]byte("a"), maxLedgerRecordBytes+2), '\n'), 0o600))
	_, _, err := LatestRecord(path, "m", "ranger")
	require.ErrorContains(t, err, "leveling_ledger_record_oversized")
}

func TestConcurrentAppendAndRotationPreserveValidRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-levels.jsonl")
	var group sync.WaitGroup
	errs := make(chan error, 21)
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			errs <- AppendRecord(path, Record{MissionID: "m", Run: string(rune('a' + index)), Level: Level{Role: "ranger", Model: "model", Effort: "high"}})
		}(i)
	}
	group.Add(1)
	go func() {
		defer group.Done()
		_, err := RotateLedger(path, 5)
		errs <- err
	}()
	group.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	records, err := ReadRecords(path)
	require.NoError(t, err)
	require.NotEmpty(t, records)
	for _, record := range records {
		require.Equal(t, LedgerSchemaVersion, record.SchemaVersion)
	}
}
