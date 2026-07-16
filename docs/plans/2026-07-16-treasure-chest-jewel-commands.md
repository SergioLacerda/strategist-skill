# Treasure Chest Jewel Commands Implementation Plan

> **REQUIRED SUB-SKILL:** Use executing-plans to implement this plan task-by-task.

**Goal:** Add `strategist treasure-chest jewel list` and `strategist treasure-chest jewel show <id>` so curated jewels (accepted/verified/deprecated) become visible again after `mine --accept`/`mine --verify`, and a single jewel's full content can be inspected.

**Architecture:** One new file, `cmd/strategist/treasure_chest_jewel.go`, adds a `jewel` group command under `treasureChestCmd` with two subcommands, `list` and `show`. Both reuse the existing `jewelEntry`/`loadJewels`/`loadGoverned` read path from `treasure_chest_jewels.go` — no new parsing or validation logic, no changes to `mine`, `add`, `remove`, `index`, or `root.go`.

**Tech Stack:** Go, `spf13/cobra` (CLI), `gopkg.in/yaml.v3` (read-only jewel parsing via existing `loadJewels`), `text/tabwriter` (table rendering), `encoding/json` (JSON rendering), `testify` (tests).

Spec: `.analysis/pending/20260716-treasure-chest-jewel-commands-design.md`

---

## Before You Start

Read these existing files — the new code must match their conventions exactly:

- `cmd/strategist/treasure_chest_jewels.go` — `jewelEntry` struct, `loadJewels`, `loadGoverned`. This is the data layer the new commands read from. Don't touch this file.
- `cmd/strategist/treasure_chest_mine.go` — the closest sibling command. `list`'s filter/sort/render logic and `show`'s single-entry lookup are adaptations of this file's `runTreasureChestMineList`/`renderMineTable`/`renderMineJSON`.
- `cmd/strategist/treasure_chest_mutate.go` — shows how two subcommands (`add`, `remove`) are grouped and registered via `init()`. `jewel` follows the same registration pattern, one level deeper (`jewel` itself is a group, not a leaf).
- `cmd/strategist/treasure_chest_mine_test.go` and `cmd/strategist/treasure_chest_test.go` — test helpers `minimalTreasureChestRoot(t)`, `resetTreasureChestFlags(t)`, `captureStdout(t, fn)`. Reuse these; do not duplicate them.
- `internal/domain/jewel_grade.go` — `JewelStatusProposed`/`Accepted`/`Verified`/`Deprecated` constants and `ValidateJewelStatus`.

---

### Task 1: Scaffold the `jewel` group command with `list` (table format only, default filter)

**Files:**
- Create: `cmd/strategist/treasure_chest_jewel.go`
- Test: `cmd/strategist/treasure_chest_jewel_test.go`

**Step 1: Write the failing test**

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const threeStatusJewelsYAML = `
schema_version: "1"
jewels:
  - id: jewel-proposed-1
    chest_id: source
    kind: pattern
    statement: "Proposed jewel statement."
    source_refs: ["source#a"]
    trust: T1
    status: proposed
    reviewed_by: agent
    score: { value: 40, reasons: ["seen once"] }
  - id: jewel-accepted-1
    chest_id: source
    kind: gap
    statement: "Accepted jewel statement."
    source_refs: ["source#b"]
    trust: T1
    status: accepted
    reviewed_by: human
    last_reviewed: "2026-07-15"
    score: { value: 60, reasons: ["seen twice"] }
  - id: jewel-deprecated-1
    chest_id: source
    kind: gap
    statement: "Deprecated jewel statement."
    source_refs: ["source#c"]
    trust: T1
    status: deprecated
    reviewed_by: human
    last_reviewed: "2026-07-10"
    score: { value: 10, reasons: ["stale"] }
`

func TestTreasureChestJewelList_DefaultExcludesDeprecated(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "jewel-accepted-1")
	assert.NotContains(t, out, "jewel-deprecated-1")
}
```

This test won't compile yet — `treasureChestJewelListCmd` and `resetTreasureChestJewelFlags` don't exist. That's expected for this step.

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/strategist/... -run TestTreasureChestJewelList_DefaultExcludesDeprecated -v`
Expected: FAIL — build error, `undefined: treasureChestJewelListCmd` (or similar)

**Step 3: Write minimal implementation**

Create `cmd/strategist/treasure_chest_jewel.go`:

```go
package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// --- treasure-chest jewel (Track: treasure-chest-jewel-commands) ---
//
// `jewel` is the read-only inspection surface over all jewels regardless of status —
// unlike `mine --list`, which is scoped to the status:proposed curation queue only.
// See .analysis/pending/20260716-treasure-chest-jewel-commands-design.md.

var (
	treasureChestJewelStatus string
	treasureChestJewelChest  string
	treasureChestJewelFormat string
)

var treasureChestJewelCmd = &cobra.Command{
	Use:   "jewel",
	Short: "Inspect jewels regardless of curation status",
}

var treasureChestJewelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List jewels, optionally filtered by status or chest",
	Long: `Lists jewels across all chests.

Without --status: shows proposed, accepted, and verified jewels (everything "alive").
deprecated jewels are excluded unless --status=all or --status=deprecated is given.`,
	RunE: runTreasureChestJewelList,
}

func init() {
	treasureChestJewelListCmd.Flags().StringVar(&treasureChestJewelStatus, "status", "", "filter by status: all, proposed, accepted, verified, or deprecated (default: all except deprecated)")
	treasureChestJewelListCmd.Flags().StringVar(&treasureChestJewelChest, "chest", "", "filter by chest id")
	treasureChestJewelListCmd.Flags().StringVar(&treasureChestJewelFormat, "format", "table", "output format: table or json")

	treasureChestJewelCmd.AddCommand(treasureChestJewelListCmd)
	treasureChestCmd.AddCommand(treasureChestJewelCmd)
}

var validJewelListStatuses = map[string]bool{
	"all":                        true,
	domain.JewelStatusProposed:   true,
	domain.JewelStatusAccepted:   true,
	domain.JewelStatusVerified:   true,
	domain.JewelStatusDeprecated: true,
}

func runTreasureChestJewelList(cmd *cobra.Command, _ []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	if treasureChestJewelStatus != "" && !validJewelListStatuses[treasureChestJewelStatus] {
		return fmt.Errorf("treasure-chest jewel list: unknown --status %q (want all, proposed, accepted, verified, or deprecated)", treasureChestJewelStatus)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest jewel list: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRoot, cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest jewel list: %w", err)
	}

	// Best-effort: governed is trust-ceiling context only, not required for listing.
	governed, govErr := loadGoverned(root)
	if govErr != nil {
		governed = nil
	}
	jewelsByChest, err := loadJewels(root, governed)
	if err != nil {
		return fmt.Errorf("treasure-chest jewel list: %w", err)
	}

	var filtered []jewelEntry
	for _, list := range jewelsByChest {
		for _, j := range list {
			if treasureChestJewelChest != "" && j.ChestID != treasureChestJewelChest {
				continue
			}
			switch treasureChestJewelStatus {
			case "":
				if j.Status == domain.JewelStatusDeprecated {
					continue
				}
			case "all":
				// no filter
			default:
				if j.Status != treasureChestJewelStatus {
					continue
				}
			}
			filtered = append(filtered, j)
		}
	}
	sort.Slice(filtered, func(i, k int) bool {
		if filtered[i].ChestID != filtered[k].ChestID {
			return filtered[i].ChestID < filtered[k].ChestID
		}
		return filtered[i].ID < filtered[k].ID
	})

	switch treasureChestJewelFormat {
	case "", "table":
		return renderJewelListTable(filtered)
	default:
		return fmt.Errorf("treasure-chest jewel list: unknown --format %q (want table or json)", treasureChestJewelFormat)
	}
}

func renderJewelListTable(jewels []jewelEntry) error {
	if len(jewels) == 0 {
		fmt.Println("[Strategist] treasure-chest jewel list: no jewels match the given filters")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "CHEST", "STATUS", "KIND", "TRUST", "SCORE", "STATEMENT"); err != nil {
		return fmt.Errorf("treasure-chest jewel list: write header: %w", err)
	}
	for _, j := range jewels {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n", j.ID, j.ChestID, j.Status, j.Kind, j.Trust, j.Score.Value, j.Statement); err != nil {
			return fmt.Errorf("treasure-chest jewel list: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest jewel list: flush: %w", err)
	}
	return nil
}
```

Add the test helper to `cmd/strategist/treasure_chest_jewel_test.go` (same file as the test above, above the test function):

```go
func resetTreasureChestJewelFlags(t *testing.T) {
	t.Helper()
	origStatus := treasureChestJewelStatus
	origChest := treasureChestJewelChest
	origFormat := treasureChestJewelFormat
	t.Cleanup(func() {
		treasureChestJewelStatus = origStatus
		treasureChestJewelChest = origChest
		treasureChestJewelFormat = origFormat
	})
	treasureChestJewelStatus = ""
	treasureChestJewelChest = ""
	treasureChestJewelFormat = "table"
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/strategist/... -run TestTreasureChestJewelList_DefaultExcludesDeprecated -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/strategist/treasure_chest_jewel.go cmd/strategist/treasure_chest_jewel_test.go
git commit -m "feat: add treasure-chest jewel list (table, default status filter)"
```

---

### Task 2: `--status` filter values (all, proposed, accepted, verified, deprecated) and invalid value rejection

**Files:**
- Modify: `cmd/strategist/treasure_chest_jewel_test.go`

**Step 1: Write the failing tests**

Append to `cmd/strategist/treasure_chest_jewel_test.go`:

```go
func TestTreasureChestJewelList_StatusAllIncludesDeprecated(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelStatus = "all"

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "jewel-accepted-1")
	assert.Contains(t, out, "jewel-deprecated-1")
}

func TestTreasureChestJewelList_ExplicitStatusFilter(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelStatus = "accepted"

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.NotContains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "jewel-accepted-1")
	assert.NotContains(t, out, "jewel-deprecated-1")
}

func TestTreasureChestJewelList_ChestFilter(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelStatus = "all"
	treasureChestJewelChest = "nonexistent-chest"

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "no jewels match")
}

func TestTreasureChestJewelList_InvalidStatusRejected(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelStatus = "bogus"

	err := treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown --status "bogus"`)
}

func TestTreasureChestJewelList_EmptyResultIsNotAnError(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelChest = "no-such-chest"

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "no jewels match the given filters")
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./cmd/strategist/... -run TestTreasureChestJewelList -v`
Expected: PASS for all five (the implementation from Task 1 already handles `all`, explicit status values, `--chest`, invalid status, and empty results — this task exists to prove it with tests, not to add new production code).

If any fail, re-check the filter `switch` in `runTreasureChestJewelList` against the test's expectations before touching test code.

**Step 3: Commit**

```bash
git add cmd/strategist/treasure_chest_jewel_test.go
git commit -m "test: cover treasure-chest jewel list status/chest filters"
```

---

### Task 3: `--format json` for `jewel list`

**Files:**
- Modify: `cmd/strategist/treasure_chest_jewel.go`
- Modify: `cmd/strategist/treasure_chest_jewel_test.go`

**Step 1: Write the failing test**

Append to `cmd/strategist/treasure_chest_jewel_test.go`:

```go
func TestTreasureChestJewelList_JSONFormat(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelStatus = "all"
	treasureChestJewelFormat = "json"

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	var decoded []jsonJewelListEntry
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	require.Len(t, decoded, 3)
	assert.Equal(t, "jewel-accepted-1", decoded[0].ID)
	assert.Equal(t, "accepted", decoded[0].Status)
}
```

Add `"encoding/json"` to the test file's import block.

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/strategist/... -run TestTreasureChestJewelList_JSONFormat -v`
Expected: FAIL — build error, `undefined: jsonJewelListEntry`

**Step 3: Write minimal implementation**

In `cmd/strategist/treasure_chest_jewel.go`, add `"encoding/json"` to the import block, then add below `renderJewelListTable`:

```go
type jsonJewelListEntry struct {
	ID           string   `json:"id"`
	ChestID      string   `json:"chest_id"`
	Status       string   `json:"status"`
	Kind         string   `json:"kind"`
	Statement    string   `json:"statement"`
	SourceRefs   []string `json:"source_refs"`
	Trust        string   `json:"trust"`
	ScoreValue   int      `json:"score_value"`
	ScoreReasons []string `json:"score_reasons,omitempty"`
}

func renderJewelListJSON(jewels []jewelEntry) error {
	out := make([]jsonJewelListEntry, 0, len(jewels))
	for _, j := range jewels {
		out = append(out, jsonJewelListEntry{
			ID:           j.ID,
			ChestID:      j.ChestID,
			Status:       j.Status,
			Kind:         j.Kind,
			Statement:    j.Statement,
			SourceRefs:   j.SourceRefs,
			Trust:        j.Trust,
			ScoreValue:   j.Score.Value,
			ScoreReasons: j.Score.Reasons,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest jewel list: encode json: %w", err)
	}
	return nil
}
```

Then update the format switch in `runTreasureChestJewelList`:

```go
	switch treasureChestJewelFormat {
	case "", "table":
		return renderJewelListTable(filtered)
	case "json":
		return renderJewelListJSON(filtered)
	default:
		return fmt.Errorf("treasure-chest jewel list: unknown --format %q (want table or json)", treasureChestJewelFormat)
	}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/strategist/... -run TestTreasureChestJewelList -v`
Expected: PASS for all six list tests

**Step 5: Commit**

```bash
git add cmd/strategist/treasure_chest_jewel.go cmd/strategist/treasure_chest_jewel_test.go
git commit -m "feat: add --format json to treasure-chest jewel list"
```

---

### Task 4: `jewel show <id>` (table format)

**Files:**
- Modify: `cmd/strategist/treasure_chest_jewel.go`
- Modify: `cmd/strategist/treasure_chest_jewel_test.go`

**Step 1: Write the failing tests**

Append to `cmd/strategist/treasure_chest_jewel_test.go`:

```go
func TestTreasureChestJewelShow_Found(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelShowCmd.RunE(treasureChestJewelShowCmd, []string{"jewel-accepted-1"}))
	})
	assert.Contains(t, out, "jewel-accepted-1")
	assert.Contains(t, out, "Accepted jewel statement.")
	assert.Contains(t, out, "source#b")
	assert.Contains(t, out, "accepted")
}

func TestTreasureChestJewelShow_NotFound(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir

	err := treasureChestJewelShowCmd.RunE(treasureChestJewelShowCmd, []string{"no-such-jewel"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `jewel "no-such-jewel" not found`)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./cmd/strategist/... -run TestTreasureChestJewelShow -v`
Expected: FAIL — build error, `undefined: treasureChestJewelShowCmd`

**Step 3: Write minimal implementation**

In `cmd/strategist/treasure_chest_jewel.go`, add the show command definition after `treasureChestJewelListCmd`:

```go
var treasureChestJewelShowCmd = &cobra.Command{
	Use:   "show <jewel-id>",
	Short: "Show a single jewel's full content",
	Args:  cobra.ExactArgs(1),
	RunE:  runTreasureChestJewelShow,
}
```

Add a `--format` flag for `show` — reuse the same `treasureChestJewelFormat` var (mutually exclusive with `list` at the command level, so sharing the flag var is safe):

In `init()`, after the existing `treasureChestJewelListCmd.Flags()...` lines:

```go
	treasureChestJewelShowCmd.Flags().StringVar(&treasureChestJewelFormat, "format", "table", "output format: table or json")

	treasureChestJewelCmd.AddCommand(treasureChestJewelShowCmd)
```

Add the lookup and render functions:

```go
func runTreasureChestJewelShow(cmd *cobra.Command, args []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	id := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest jewel show: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRoot, cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest jewel show: %w", err)
	}

	governed, govErr := loadGoverned(root)
	if govErr != nil {
		governed = nil
	}
	jewelsByChest, err := loadJewels(root, governed)
	if err != nil {
		return fmt.Errorf("treasure-chest jewel show: %w", err)
	}

	var found *jewelEntry
	for _, list := range jewelsByChest {
		for i := range list {
			if list[i].ID == id {
				found = &list[i]
				break
			}
		}
	}
	if found == nil {
		return fmt.Errorf("treasure-chest jewel show: jewel %q not found", id)
	}

	switch treasureChestJewelFormat {
	case "", "table":
		return renderJewelShowTable(*found)
	case "json":
		return renderJewelShowJSON(*found)
	default:
		return fmt.Errorf("treasure-chest jewel show: unknown --format %q (want table or json)", treasureChestJewelFormat)
	}
}

func renderJewelShowTable(j jewelEntry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	rows := [][2]string{
		{"id", j.ID},
		{"chest_id", j.ChestID},
		{"kind", j.Kind},
		{"status", j.Status},
		{"trust", j.Trust},
		{"statement", j.Statement},
		{"source_refs", fmt.Sprintf("%v", j.SourceRefs)},
		{"reviewed_by", j.ReviewedBy},
		{"last_reviewed", j.LastReviewed},
		{"score.value", fmt.Sprintf("%d", j.Score.Value)},
		{"score.reasons", fmt.Sprintf("%v", j.Score.Reasons)},
		{"applicability.scope", fmt.Sprintf("%v", j.Applicability.Scope)},
		{"applicability.applies_when", fmt.Sprintf("%v", j.Applicability.AppliesWhen)},
		{"applicability.avoid_when", fmt.Sprintf("%v", j.Applicability.AvoidWhen)},
		{"verification.evidence_refs", fmt.Sprintf("%v", j.Verification.EvidenceRefs)},
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", r[0], r[1]); err != nil {
			return fmt.Errorf("treasure-chest jewel show: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest jewel show: flush: %w", err)
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./cmd/strategist/... -run TestTreasureChestJewelShow -v`
Expected: PASS for `TestTreasureChestJewelShow_Found` and `TestTreasureChestJewelShow_NotFound`

Note: `renderJewelShowJSON` doesn't exist yet — this is expected; it's added in Task 5. Task 4's tests only exercise the table path, so they'll pass without it as long as the `case "json":` branch isn't hit by the test inputs (it isn't — both tests above use the default format).

**Step 5: Commit**

```bash
git add cmd/strategist/treasure_chest_jewel.go cmd/strategist/treasure_chest_jewel_test.go
git commit -m "feat: add treasure-chest jewel show (table format)"
```

---

### Task 5: `jewel show --format json`

**Files:**
- Modify: `cmd/strategist/treasure_chest_jewel.go`
- Modify: `cmd/strategist/treasure_chest_jewel_test.go`

**Step 1: Write the failing test**

Append to `cmd/strategist/treasure_chest_jewel_test.go`:

```go
func TestTreasureChestJewelShow_JSONFormat(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelFormat = "json"

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelShowCmd.RunE(treasureChestJewelShowCmd, []string{"jewel-accepted-1"}))
	})
	var decoded jsonJewelShowEntry
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	assert.Equal(t, "jewel-accepted-1", decoded.ID)
	assert.Equal(t, []string{"source#b"}, decoded.SourceRefs)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/strategist/... -run TestTreasureChestJewelShow_JSONFormat -v`
Expected: FAIL — build error, `undefined: jsonJewelShowEntry` / `renderJewelShowJSON`

**Step 3: Write minimal implementation**

In `cmd/strategist/treasure_chest_jewel.go`, add after `renderJewelShowTable`:

```go
type jsonJewelShowEntry struct {
	ID            string             `json:"id"`
	ChestID       string             `json:"chest_id"`
	Kind          string             `json:"kind"`
	Status        string             `json:"status"`
	Trust         string             `json:"trust"`
	Statement     string             `json:"statement"`
	SourceRefs    []string           `json:"source_refs"`
	ReviewedBy    string             `json:"reviewed_by"`
	LastReviewed  string             `json:"last_reviewed,omitempty"`
	Score         jewelScore         `json:"score"`
	Applicability jewelApplicability `json:"applicability"`
	Verification  jewelVerification  `json:"verification"`
}

func renderJewelShowJSON(j jewelEntry) error {
	out := jsonJewelShowEntry{
		ID:            j.ID,
		ChestID:       j.ChestID,
		Kind:          j.Kind,
		Status:        j.Status,
		Trust:         j.Trust,
		Statement:     j.Statement,
		SourceRefs:    j.SourceRefs,
		ReviewedBy:    j.ReviewedBy,
		LastReviewed:  j.LastReviewed,
		Score:         j.Score,
		Applicability: j.Applicability,
		Verification:  j.Verification,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest jewel show: encode json: %w", err)
	}
	return nil
}
```

This requires `jewelScore`, `jewelApplicability`, `jewelVerification` (defined in `treasure_chest_jewels.go`) to have JSON-unmarshalable field names — check that file: they currently only have `yaml` tags, no `json` tags. Since Go's `encoding/json` falls back to the Go field name (`Value`, `Reasons`, `Scope`, `AppliesWhen`, `AvoidWhen`, `EvidenceRefs`) when no `json` tag is present, this works but produces `PascalCase` keys nested under `score`/`applicability`/`verification` instead of `snake_case`. That's inconsistent with the rest of this JSON output (`chest_id`, `source_refs`, etc.).

Fix it at the source — add `json` tags to the three structs in `cmd/strategist/treasure_chest_jewels.go`:

```go
type jewelScore struct {
	Value   int      `yaml:"value" json:"value"`
	Reasons []string `yaml:"reasons" json:"reasons"`
}

type jewelApplicability struct {
	Scope       []string `yaml:"scope" json:"scope"`
	AppliesWhen []string `yaml:"applies_when" json:"applies_when"`
	AvoidWhen   []string `yaml:"avoid_when" json:"avoid_when"`
}

type jewelVerification struct {
	EvidenceRefs []string `yaml:"evidence_refs" json:"evidence_refs"`
}
```

This is a one-line-per-field additive change (adds `json` tags, doesn't remove or rename `yaml` tags) — it does not affect `loadJewels`'s YAML parsing or any existing caller.

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/strategist/... -run TestTreasureChestJewelShow -v`
Expected: PASS for both show tests

**Step 5: Run the full package test suite**

Run: `go test ./cmd/strategist/... -v 2>&1 | tail -40`
Expected: all tests PASS, no regressions in `treasure_chest_mine_test.go` or `treasure_chest_test.go` (the `json` tag addition is additive-only)

**Step 6: Commit**

```bash
git add cmd/strategist/treasure_chest_jewel.go cmd/strategist/treasure_chest_jewels.go cmd/strategist/treasure_chest_jewel_test.go
git commit -m "feat: add --format json to treasure-chest jewel show"
```

---

### Task 6: Update `docs/cli-reference.md`

**Files:**
- Modify: `docs/cli-reference.md`

**Step 1: Add the new commands to the `index`/`mine` usage block**

Find this block (around line 433-438):

```
strategist treasure-chest index [--include-historical]
strategist treasure-chest mine --list [--format table|json]
strategist treasure-chest mine --accept <jewel-id>
strategist treasure-chest mine --verify <jewel-id> --evidence <ref>
strategist treasure-chest mine --deprecate <jewel-id>
strategist treasure-chest mine --migrate-status
```

Append two lines directly after it:

```
strategist treasure-chest jewel list [--status all|proposed|accepted|verified|deprecated] [--chest <chest-id>] [--format table|json]
strategist treasure-chest jewel show <jewel-id> [--format table|json]
```

**Step 2: Add a `jewel list` / `jewel show` subsection**

Find the end of the `mine` bullet list (ends with `- --migrate-status — see Migration below.` around line 470) and insert a new subsection directly after it, before the `treasure-chest scan` contract paragraph:

```markdown
**`jewel`** is the read-only inspection surface over all jewels regardless of status —
unlike `mine --list` (scoped to the `status: proposed` curation queue only):

- `list [--status all|proposed|accepted|verified|deprecated] [--chest <chest-id>] [--format table|json]`
  — without `--status`, shows `proposed` + `accepted` + `verified` (excludes
  `deprecated`); `--status all` includes `deprecated`; `--chest` filters by chest id,
  combinable with `--status`. Sorted by `(chest_id, id)`, same as `mine --list`.
- `show <jewel-id> [--format table|json]` — prints every field of a single jewel
  (`statement`, `source_refs`, `trust`, `score`, `applicability`, `verification`,
  etc.). Unknown id: error, non-zero exit.
```

**Step 3: Update the "Implemented" note**

Find the sentence near the end of the Jewels section (around line 542-545):

```
**Implemented**: `.strategist/jewels.yaml` exists and is loaded via `loadJewels`
(`cmd/strategist/treasure_chest_jewels.go`); non-deprecated jewel counts are shown in the
`treasure-chest` list's `JEWELS` column and JSON output (`cmd/strategist/treasure_chest.go`);
removing a chest cascades to mark its jewels `deprecated` (`markJewelsDeprecatedForChest` in
`treasure_chest_yaml_node.go`); the `jewel_generation` and `jewel_retrieval` contract blocks
```

Add a clause before the final `;`-joined item (check the sentence's continuation past what's quoted above and append there — read the full sentence in the file first since it continues beyond this excerpt):

```
; `treasure-chest jewel list`/`jewel show` (`cmd/strategist/treasure_chest_jewel.go`)
expose all jewels regardless of status for inspection, independent of `mine`'s curation
queue
```

**Step 4: Verify markdown renders sanely**

Run: `grep -n "jewel list\|jewel show" docs/cli-reference.md`
Expected: shows the new usage lines and subsection, no broken structure

**Step 5: Commit**

```bash
git add docs/cli-reference.md
git commit -m "docs: document treasure-chest jewel list/show commands"
```

---

### Task 7: Full regression pass

**Step 1: Build**

Run: `go build ./...`
Expected: no errors

**Step 2: Full test suite**

Run: `go test $(go list ./... | grep -v '/testutil')`
Expected: all packages `ok`

**Step 3: Spec tests**

Run: `go test -tags=spec ./tests/spec/...`
Expected: `ok` (this feature doesn't touch any spec-tested contract file, so this should be unaffected — run it anyway to confirm no accidental drift)

**Step 4: Manual smoke test**

```bash
go build -o bin/strategist ./cmd/strategist
cd /tmp && rm -rf jewel-cmd-smoke && mkdir jewel-cmd-smoke && cd jewel-cmd-smoke
/home/sergio/dev/strategist-skill/bin/strategist install >/dev/null 2>&1
/home/sergio/dev/strategist-skill/bin/strategist treasure-chest index
/home/sergio/dev/strategist-skill/bin/strategist treasure-chest jewel list
/home/sergio/dev/strategist-skill/bin/strategist treasure-chest jewel list --status all --format json
/home/sergio/dev/strategist-skill/bin/strategist treasure-chest jewel show <some-id-from-the-list-output>
```

Expected: `jewel list` shows proposed jewels from the fresh index (no accepted/verified/deprecated exist yet in a brand-new install, so this mostly confirms the command runs cleanly); `jewel show` prints the full detail block for a real id from the list.

**Step 5: Commit (if smoke test caught anything to fix)**

Only if Step 4 revealed a bug — otherwise this task has nothing to commit.

---

## Out of Scope (per spec)

- Per-jewel staleness/freshness tracking.
- Any change to `discovery`/`scan`/`index` behavior.
- Any change to `mine`'s existing flags or behavior.
