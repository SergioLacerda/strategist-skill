package mission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var numberedADRName = regexp.MustCompile(`^(\d+)-.*\.md$`)

var slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

// adrSlug turns a title into a file-name slug; an empty result falls back to
// the mission id.
func adrSlug(slug, missionID string) string {
	clean := strings.Trim(slugUnsafe.ReplaceAllString(strings.ToLower(slug), "-"), "-")
	if clean == "" {
		return missionID
	}
	return clean
}

// nextADRFilename scans dir for numbered ADRs (^\d+-.*\.md$) and continues
// the highest sequence, zero-padded to the same width; an empty or
// non-numbered directory falls back to <mission_id>-adr.md. reserved lists
// file names already reserved for other missions in the same directory so two
// missions cannot reserve the same number.
func nextADRFilename(dir string, reserved []string, missionID, slug string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("adr_target_unreadable: scan %s: %w", dir, err)
	}
	names := append([]string(nil), reserved...)
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	highest, width := highestADRNumber(names)
	if width == 0 {
		return missionID + "-adr.md", nil
	}
	return fmt.Sprintf("%0*d-%s.md", width, highest+1, adrSlug(slug, missionID)), nil
}

func highestADRNumber(names []string) (highest, width int) {
	for _, name := range names {
		m := numberedADRName.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if n >= highest {
			highest = n
		}
		width = max(width, len(m[1]))
	}
	return highest, width
}

// reservedNamesIn lists the file names other missions reserved inside the
// workspace-relative directory rel. Unreadable or foreign records are
// ignored: this only widens the numbering scan, it never gates it.
func reservedNamesIn(strategistRoot, missionID, rel string) []string {
	entries, err := os.ReadDir(filepath.Join(strategistRoot, "missions", "side-quests"))
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if entry.Name() == missionID+".json" {
			continue
		}
		if name, ok := reservedNameOf(filepath.Join(strategistRoot, "missions", "side-quests", entry.Name()), rel); ok {
			names = append(names, name)
		}
	}
	return names
}

func reservedNameOf(recordPath, rel string) (string, bool) {
	raw, err := os.ReadFile(recordPath) //nolint:gosec // entry of the runtime's own side-quests directory
	if err != nil {
		return "", false
	}
	var other AcceptedSideQuest
	if json.Unmarshal(raw, &other) != nil || other.ReservedPath == "" || filepath.ToSlash(filepath.Dir(other.ReservedPath)) != rel {
		return "", false
	}
	return filepath.Base(other.ReservedPath), true
}
