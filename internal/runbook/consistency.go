package runbook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Finding severities. A skip finding means the sidecar must not be offered;
// a warn finding means it stays usable but is inconsistent and must be
// reported.
const (
	SeveritySkip = "skip"
	SeverityWarn = "warn"
)

// Finding codes. They are stable: callers print them and tests pin them.
const (
	CodeSourceDocMissing     = "source_doc_missing"
	CodeSourceDocNotMarkdown = "source_doc_not_markdown"
	CodeIDStemMismatch       = "runbook_id_stem_mismatch"
)

const sidecarFileSuffix = ".runbook.yaml"

// Finding is one consistency problem of a sidecar that ParseSidecar accepts.
type Finding struct {
	Code     string
	Severity string
	Detail   string
}

// CheckSidecarConsistency checks what the schema parser deliberately does not:
// that the referenced markdown exists and is markdown, and that runbook_id
// follows the file stem. It is kept beside ParseSidecar so the parser's
// documented contract (docs/runbooks/README.md) does not change.
// sidecarPath is project-relative (or absolute); projectRoot resolves
// source_doc.
func CheckSidecarConsistency(sidecarPath string, rb Runbook, projectRoot string) []Finding {
	var findings []Finding
	if stem := strings.TrimSuffix(filepath.Base(sidecarPath), sidecarFileSuffix); rb.RunbookID != stem {
		findings = append(findings, Finding{
			Code: CodeIDStemMismatch, Severity: SeverityWarn,
			Detail: fmt.Sprintf("runbook_id %q differs from the file stem %q", rb.RunbookID, stem),
		})
	}
	return append(findings, checkSourceDoc(rb.SourceDoc, projectRoot)...)
}

func checkSourceDoc(sourceDoc, projectRoot string) []Finding {
	if !filepath.IsLocal(filepath.FromSlash(sourceDoc)) {
		return []Finding{{Code: CodeSourceDocMissing, Severity: SeveritySkip,
			Detail: fmt.Sprintf("source_doc %q is not a project-relative path", sourceDoc)}}
	}
	info, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(sourceDoc)))
	if err != nil || info.IsDir() {
		return []Finding{{Code: CodeSourceDocMissing, Severity: SeveritySkip,
			Detail: fmt.Sprintf("source_doc %q does not exist", sourceDoc)}}
	}
	if !strings.HasSuffix(sourceDoc, ".md") {
		return []Finding{{Code: CodeSourceDocNotMarkdown, Severity: SeverityWarn,
			Detail: fmt.Sprintf("source_doc %q is not a markdown file", sourceDoc)}}
	}
	return nil
}
