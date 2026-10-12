package dojo

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
)

// Audit verdicts.
const (
	VerdictAgree    = "agree"
	VerdictDisagree = "disagree"
)

// AuditEntry records one review of a generated case: does the mutation really
// violate what it claims. A disagreement excludes the case from the metrics; an
// audit never relabels a case.
type AuditEntry struct {
	CaseID  string `json:"case_id"`
	Verdict string `json:"verdict"`
	Note    string `json:"note,omitempty"`
	By      string `json:"by"`
	At      string `json:"at"`
}

// SampleForAudit picks n generated negatives, spread deterministically over the
// operators: the same cases and seed always give the same sample. Stage 2 cases
// come first because they count only once audited.
func SampleForAudit(cases []calibration.Case, n int, seed int64) []calibration.Case {
	var pool []calibration.Case
	for _, c := range cases {
		if c.Operator != "" && c.Kind != calibration.KindNeutral {
			pool = append(pool, c)
		}
	}
	rank := func(c calibration.Case) string {
		return fmt.Sprintf("%d|%x", 2-c.Stage, sha256.Sum256([]byte(fmt.Sprintf("%d|%s", seed, c.ID))))
	}
	sort.Slice(pool, func(i, j int) bool { return rank(pool[i]) < rank(pool[j]) })
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:max(n, 0)]
}

// RecordAudit appends one audit entry. The reviewer is required.
func RecordAudit(path string, entry AuditEntry, now time.Time) error {
	if entry.CaseID == "" || strings.TrimSpace(entry.By) == "" {
		return errors.New("dojo: an audit entry needs a case id and a reviewer")
	}
	if entry.Verdict != VerdictAgree && entry.Verdict != VerdictDisagree {
		return fmt.Errorf("dojo: audit verdict %q must be agree or disagree", entry.Verdict)
	}
	entry.At = now.UTC().Format(time.RFC3339)
	return appendJSONL(path, entry)
}

// LoadAudit reads the audit log and turns it into the metrics filter: agreed
// cases are audited, disagreed ones are excluded. The last entry per case wins.
func LoadAudit(path string) (calibration.Filter, []AuditEntry, error) {
	filter := calibration.Filter{Audited: map[string]bool{}, Excluded: map[string]bool{}}
	var entries []AuditEntry
	bad := 0
	err := readJSONL(path, func(raw []byte) bool {
		var entry AuditEntry
		if json.Unmarshal(raw, &entry) != nil || entry.CaseID == "" {
			return false
		}
		entries = append(entries, entry)
		filter.Audited[entry.CaseID] = entry.Verdict == VerdictAgree
		filter.Excluded[entry.CaseID] = entry.Verdict == VerdictDisagree
		return true
	}, &bad)
	return filter, entries, err
}

// Proposal is an operator an agent proposes. It waits for a reviewed change to
// the catalog; nothing here adds an operator.
type Proposal struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Reason      string `json:"reason,omitempty"`
	Status      string `json:"status"`
	By          string `json:"proposed_by"`
	At          string `json:"at"`
}

// AppendProposal records a pending operator proposal under mutations/.
func AppendProposal(basePath string, p Proposal, now time.Time) error {
	if !validScenarioName(p.Name) || strings.TrimSpace(p.Description) == "" {
		return errors.New("dojo: a proposal needs a plain operator name and a description")
	}
	if _, exists := calibration.ByName(p.Name); exists {
		return fmt.Errorf("dojo: operator %q already exists", p.Name)
	}
	p.Status, p.At = "pending", now.UTC().Format(time.RFC3339)
	return appendJSONL(filepath.Join(basePath, "dojo", DirMutations, "proposals.jsonl"), p)
}
