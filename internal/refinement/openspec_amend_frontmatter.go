package refinement

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// frontmatterHeader returns the text between the opening and closing "---" lines of
// content, or "" when it has no frontmatter.
func frontmatterHeader(content []byte) string {
	text := string(content)
	if !strings.HasPrefix(text, "---\n") {
		return ""
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return ""
	}
	return text[4 : 4+end]
}

// frontmatterValue is the value of a top-level scalar key in the frontmatter, or "".
func frontmatterValue(content []byte, key string) string {
	for _, line := range strings.Split(frontmatterHeader(content), "\n") {
		if strings.HasPrefix(line, key+":") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, key+":")), `"'`)
		}
	}
	return ""
}

// priorAmendmentEntries returns the entries of the `amendments:` list already in
// content's frontmatter, so a replacement file keeps the list append-only.
func priorAmendmentEntries(content []byte) string {
	var entries []string
	collecting := false
	for _, line := range strings.Split(frontmatterHeader(content), "\n") {
		switch {
		case line == "amendments:":
			collecting = true
		case collecting && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "-")):
			entries = append(entries, line)
		default:
			collecting = false
		}
	}
	if len(entries) == 0 {
		return ""
	}
	return strings.Join(entries, "\n") + "\n"
}

// withAmendments stamps next with the amendments list of previous plus one new
// entry. The digest of next itself lives only in the manifest: a file cannot
// contain its own digest.
func withAmendments(next, previous []byte, number int, changeID, ref string, at time.Time, plan *amendmentPlan) []byte {
	entry := fmt.Sprintf("  - amendment: %d\n    change_id: %s\n    at: %s\n    authorization_ref: %s\n    derived_from: %s\n    supersedes_mission_id: %s\n    source_digest: %s\n    package_digest: %s\n    reason: %s\n    disposition: %s\n    previous_sha256: %s\n",
		number, changeID, at.Format(time.RFC3339), strconv.Quote(ref), plan.input.Amends, plan.input.SupersedesMissionID,
		plan.analysisSHA, plan.packageSHA, plan.reason, plan.disposition, digest(previous))
	block := "amendments:\n" + priorAmendmentEntries(previous) + entry
	header := frontmatterHeader(next)
	if header == "" {
		return []byte("---\n" + block + "---\n\n" + string(next))
	}
	body := string(next)[4+len(header):]
	return []byte("---\n" + header + "\n" + block + strings.TrimPrefix(body, "\n"))
}
