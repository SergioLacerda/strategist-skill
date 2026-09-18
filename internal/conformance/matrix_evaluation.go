package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// Evaluate checks that every selected client surface is available and returns
// a deterministic structural report. Live rows are reported as pending unless
// callers separately provide successful probe evidence.
func (m Matrix) Evaluate(availableClients map[string]bool) (Report, error) {
	if err := m.Validate(); err != nil {
		return Report{}, err
	}
	if err := ensureClientsAvailable(m.Clients, availableClients); err != nil {
		return Report{}, err
	}
	results := evaluateRows(m.Rows)
	sort.Slice(results, func(i, j int) bool { return results[i].RowID < results[j].RowID })
	digest, err := digestMatrix(m)
	if err != nil {
		return Report{}, fmt.Errorf("conformance matrix: digest: %w", err)
	}
	return Report{SchemaVersion: m.SchemaVersion, MatrixDigest: digest, ClientCount: len(m.Clients), RowCount: len(m.Rows), Results: results}, nil
}

func ensureClientsAvailable(clients []Client, available map[string]bool) error {
	for _, client := range clients {
		if !available[client.ID] {
			return fmt.Errorf("conformance matrix: selected client %q is unavailable", client.ID)
		}
	}
	return nil
}

func evaluateRows(rows []Row) []RowResult {
	results := make([]RowResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, evaluateRow(row))
	}
	return results
}

func evaluateRow(row Row) RowResult {
	if row.EvidenceTier == EvidenceLive {
		return RowResult{RowID: row.ID, Client: row.Client, Reason: "live_probe_required"}
	}
	return RowResult{RowID: row.ID, Client: row.Client, Passed: true, Reason: "structural_evidence_verified"}
}

// EvaluateLiveProbe consumes explicit probe evidence for a live row. A probe
// is successful only when it reports certified evidence; unknown, unsupported,
// failed, and blocked states remain non-passing and cannot activate a row.
func EvaluateLiveProbe(row Row, observed EvidenceState) (RowResult, error) {
	if row.EvidenceTier != EvidenceLive {
		return RowResult{}, fmt.Errorf("conformance live probe: row %q is not a live-evidence row", row.ID)
	}
	if !validState(observed) {
		return RowResult{}, fmt.Errorf("conformance live probe: unknown observed state %q", observed)
	}
	passed := observed == StateCertified && row.ExpectedState == StateCertified
	reason := "live_probe_not_ready"
	if passed {
		reason = "live_probe_verified"
	}
	return RowResult{RowID: row.ID, Client: row.Client, Passed: passed, Reason: reason}, nil
}

// CompareIdentity compares fields that transport adapters must not rewrite.
func CompareIdentity(expected, actual Row) error {
	if expected.Client != actual.Client || expected.Role != actual.Role || expected.Slot != actual.Slot || expected.ProviderMode != actual.ProviderMode || expected.EnvelopeVersion != actual.EnvelopeVersion || expected.EvidenceTier != actual.EvidenceTier || expected.ExpectedState != actual.ExpectedState || expected.ReasonCode != actual.ReasonCode || expected.AuthorityOwner != actual.AuthorityOwner {
		return fmt.Errorf("conformance identity mismatch: expected=%s actual=%s", rowIdentity(expected), rowIdentity(actual))
	}
	return nil
}

// MarshalReport emits stable JSON suitable for CI artifacts.
func MarshalReport(report Report) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("conformance report: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func digestMatrix(matrix Matrix) (string, error) {
	data, err := json.Marshal(matrix)
	if err != nil {
		return "", fmt.Errorf("encode matrix: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func rowIdentity(row Row) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", row.Client, row.Role, row.Slot, row.ProviderMode, row.EnvelopeVersion, row.ExpectedState, row.ReasonCode)
}
