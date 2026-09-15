package telemetry

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// FlushOutcomeBuffer moves buffered entries from bufferPath (outcomes.tmp)
// into outcomesPath (outcomes.jsonl), skipping any whose mission_id already
// exists in outcomesPath. It replaces the raw `cat >> ; truncate`
// flush_procedure with a version that cannot introduce a duplicate outcome
// record when a previous flush was interrupted between the append and the
// truncate (ADR-0004: "two paths to the same data can cause duplicates").
// A missing buffer file is not an error — it means the buffer is empty.
// The buffer is only cleared after every line has been processed, so a
// failure mid-flush leaves it intact for the next mission's retry.
// Malformed buffered lines are skipped (logged, non-blocking) rather than
// aborting the flush, consistent with the learning contract's non-blocking
// invariant.
func FlushOutcomeBuffer(bufferPath, outcomesPath string) (flushed int, err error) {
	data, err := readOutcomeBuffer(bufferPath)
	if err != nil {
		return 0, err
	}
	if data == nil {
		return 0, nil
	}

	flushed, err = flushOutcomeBufferData(data, outcomesPath)
	if err != nil {
		return flushed, err
	}
	return flushed, truncateOutcomeBuffer(bufferPath)
}

func readOutcomeBuffer(bufferPath string) ([]byte, error) {
	data, err := os.ReadFile(bufferPath) //nolint:gosec // G304: buffer path is owned by the Strategist runtime memory domain
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read outcomes buffer: %w", err)
	}
	return data, nil
}

func flushOutcomeBufferData(data []byte, outcomesPath string) (int, error) {
	flushed := 0
	scanner := newJSONLScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		if appendBufferedOutcomeLine(outcomesPath, scanner.Text()) {
			flushed++
		}
	}
	return flushed, jsonlScannerErr(scanner, "scan outcomes buffer")
}

func appendBufferedOutcomeLine(outcomesPath, line string) bool {
	if line == "" {
		return false
	}
	appended, err := AppendOutcomeLine(outcomesPath, line)
	if err != nil {
		slog.Warn("outcome flush: skipping invalid buffered line (non-blocking)", "error", err)
		return false
	}
	return appended
}

func truncateOutcomeBuffer(bufferPath string) error {
	if err := os.Truncate(bufferPath, 0); err != nil {
		return fmt.Errorf("truncate outcomes buffer: %w", err)
	}
	return nil
}
