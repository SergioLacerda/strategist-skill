package install

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenerateMissionID returns a date-prefixed mission ID from slug.
// If a file with the same slug already exists in pending/, refined/, or archived/
// under basePath, a short random suffix is appended to avoid collision.
func GenerateMissionID(basePath, slug string, now time.Time) string {
	date := now.Format("20060102")
	base := date + "-" + sanitizeSlug(slug)
	if !missionIDCollides(basePath, base) {
		return base
	}
	// Collision: append a 4-char hex suffix derived from a random source.
	suffix := fmt.Sprintf("%04x", rand.New(rand.NewSource(now.UnixNano())).Intn(0xffff)) //nolint:gosec
	candidate := base + "-" + suffix
	// If collision persists (extremely rare), return the suffixed form regardless.
	return candidate
}

func sanitizeSlug(slug string) string {
	slug = strings.ToLower(slug)
	var b strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// missionIDCollides checks whether any artifact named after id already exists
// in the three canonical analysis directories.
func missionIDCollides(basePath, id string) bool {
	dirs := []string{"pending", "refined", "archived"}
	for _, dir := range dirs {
		pattern := filepath.Join(basePath, dir, id+"*")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && info != nil {
				return true
			}
		}
	}
	return false
}
