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
		switch {
		case isSlugRune(r):
			b.WriteRune(r)
		case isSlugSeparator(r):
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func isSlugRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
}

func isSlugSeparator(r rune) bool {
	return r == ' ' || r == '_'
}

// missionIDCollides checks whether any artifact named after id already exists
// in the three canonical analysis directories.
func missionIDCollides(basePath, id string) bool {
	for _, pattern := range missionIDPatterns(basePath, id) {
		if patternCollides(pattern) {
			return true
		}
	}
	return false
}

func missionIDPatterns(basePath, id string) []string {
	return []string{
		filepath.Join(basePath, "pending", id+"*"),
		filepath.Join(basePath, "refined", id+"*"),
		filepath.Join(basePath, "archived", id+"*"),
	}
}

func patternCollides(pattern string) bool {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	for _, match := range matches {
		if pathExists(match) {
			return true
		}
	}
	return false
}

func pathExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info != nil
}
