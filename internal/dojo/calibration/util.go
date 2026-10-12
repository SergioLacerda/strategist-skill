package calibration

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
)

func sortedKeys(fields Fields) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (f Fields) clone() Fields {
	out := make(Fields, len(f))
	for key, value := range f {
		out[key] = value
	}
	return out
}

// pickIndexes chooses ceil(fraction*n) of n positions deterministically from the
// seed and key, so a graded level is a stable subset of the next level up.
func pickIndexes(seed int64, key string, n int, fraction float64) map[int]bool {
	type ranked struct {
		index int
		rank  uint64
	}
	items := make([]ranked, n)
	for i := range items {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", key, i, seed)))
		items[i] = ranked{i, binary.BigEndian.Uint64(sum[:8])}
	}
	sort.Slice(items, func(a, b int) bool { return items[a].rank < items[b].rank })
	take := int(float64(n)*fraction + 0.999999)
	if take > n {
		take = n
	}
	chosen := make(map[int]bool, take)
	for _, item := range items[:take] {
		chosen[item.index] = true
	}
	return chosen
}
