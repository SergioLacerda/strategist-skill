package dojo

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
)

// loadBefores reads the before snapshots of a package's amendments: the tasks
// of the state a human asked to revise, with the package's own analysis. A
// snapshot without tasks, an unreadable one, or one identical to the current
// package is skipped.
func loadBefores(item scanned, reader FieldReader) []Before {
	root := filepath.Join(item.dir, ".amendments")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var out []Before
	for _, e := range entries {
		seq, convErr := strconv.Atoi(e.Name())
		if !e.IsDir() || convErr != nil {
			continue
		}
		if before, ok := readBefore(item, filepath.Join(root, e.Name()), seq, reader); ok {
			out = append(out, before)
		}
	}
	return out
}

func readBefore(item scanned, snapshot string, seq int, reader FieldReader) (Before, bool) {
	tasks := filepath.Join(snapshot, "tasks.md")
	if _, err := os.Stat(tasks); err != nil {
		return Before{}, false
	}
	fields, err := reader.Package(tasks, filepath.Join(item.dir, "analysis.md"))
	if err != nil || len(fields) == 0 {
		return Before{}, false
	}
	digest := calibration.Digest(fields)
	if digest == item.entry.Digest {
		return Before{}, false
	}
	return Before{Entry: item.entry.Name, Digest: digest, Seq: seq, Fields: fields}, true
}
