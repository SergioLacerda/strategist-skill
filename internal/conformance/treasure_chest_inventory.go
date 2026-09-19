package conformance

import (
	"fmt"
	"time"
)

// ImportDisposition classifies an external direct importer without claiming delegation.
type ImportDisposition string

const (
	// ImportMediated indicates the import is mediated via a reviewed delegate path.
	ImportMediated ImportDisposition = "mediated"
	// ImportReviewedException indicates the import has a time-bounded reviewed staging exception.
	ImportReviewedException ImportDisposition = "reviewed_exception"
	// ImportViolation indicates an unreviewed, invalid, or expired direct import.
	ImportViolation ImportDisposition = "violation"
)

// ImportException is a time-bounded reviewed staging exception or delegate path.
type ImportException struct {
	Importer string
	Owner    string
	Reason   string
	Scope    string
	ReviewBy time.Time
	Mediated bool
}

// ImportClassification gives a stable, reportable disposition for one importer.
type ImportClassification struct {
	Importer    string
	Disposition ImportDisposition
	Reason      string
}

// ClassifyTreasureChestImporters fails unknown and expired direct imports closed.
func ClassifyTreasureChestImporters(importers []string, exceptions []ImportException, now time.Time) []ImportClassification {
	byImporter := make(map[string]ImportException, len(exceptions))
	for _, exception := range exceptions {
		byImporter[exception.Importer] = exception
	}
	classifications := make([]ImportClassification, 0, len(importers))
	for _, importer := range importers {
		classifications = append(classifications, classifyImporter(importer, byImporter[importer], now))
	}
	return classifications
}

func classifyImporter(importer string, exception ImportException, now time.Time) ImportClassification {
	if exception.Importer == "" {
		return ImportClassification{Importer: importer, Disposition: ImportViolation, Reason: "unreviewed_direct_import"}
	}
	if exception.Mediated {
		return ImportClassification{Importer: importer, Disposition: ImportMediated, Reason: "delegate_path_reviewed"}
	}
	if exception.Owner == "" || exception.Reason == "" || exception.Scope == "" || exception.ReviewBy.IsZero() {
		return ImportClassification{Importer: importer, Disposition: ImportViolation, Reason: "exception_incomplete"}
	}
	if !exception.ReviewBy.After(now) {
		return ImportClassification{Importer: importer, Disposition: ImportViolation, Reason: "exception_expired"}
	}
	return ImportClassification{Importer: importer, Disposition: ImportReviewedException, Reason: "staging_exception_reviewed"}
}

// MigrationRecord carries the evidence required before changing ownership stages.
type MigrationRecord struct {
	FromStage          BoundaryStage
	ToStage            BoundaryStage
	SourceIdentity     string
	TargetIdentity     string
	CommandParity      bool
	APICompatible      bool
	ProvenanceVerified bool
	MigrationComplete  bool
	RollbackRef        string
}

// EvaluateMigration prevents partially evidenced transitions from mutating legacy data.
func EvaluateMigration(record MigrationRecord) BoundaryResult {
	if record.FromStage == record.ToStage || record.SourceIdentity == "" || record.TargetIdentity == "" || record.RollbackRef == "" {
		return boundaryResult(BoundaryBlocked, "migration_record_incomplete")
	}
	if !record.CommandParity || !record.APICompatible || !record.ProvenanceVerified || !record.MigrationComplete {
		return boundaryResult(BoundaryBlocked, "migration_evidence_incomplete")
	}
	return BoundaryResult{Status: BoundaryReady, Reason: fmt.Sprintf("migration_ready:%s_to_%s", record.FromStage, record.ToStage), LegacyDataPreserved: true}
}
