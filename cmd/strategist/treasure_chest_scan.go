package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// --- SQ-010 (Track T-F): treasure-chest scan ---
//
// Mines .analysis/refined/**/tasks.md and .analysis/done/** (never pending/ or archived/
// reports) for recurring clusters (2+ missions sharing 2+ tags) and open gaps (side quests
// still status: sq_pending), per .analysis/refined/bau-tesouro-sq003-004-007/design.md and
// .analysis/refined/bau-tesouro-sq010-scan-runtime/design.md. Lexical/tag matching only — no
// embeddings, no semantic retrieval.

var treasureChestScanDryRun bool

var treasureChestScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Mine .analysis/refined and .analysis/done for recurring clusters and open gaps",
	Long: `Scan mission history for recurring themes and unresolved side quests.

Input: <base_path>/refined/**/tasks.md and <base_path>/done/** only. Never reads
<base_path>/pending/ or <base_path>/archived/ reports.

Method: lexical/tag matching only. No embeddings, no vector index.

Output: .strategist/treasure/clusters/ and .strategist/treasure/gaps/, regenerated from
scratch on every run (safe to delete — see docs/configuration.md § Storage Domain).

Use --dry-run to preview output without writing to disk.`,
	RunE: runTreasureChestScan,
}

func init() {
	treasureChestScanCmd.Flags().BoolVar(&treasureChestScanDryRun, "dry-run", false, "print what would be written without touching disk")
	treasureChestCmd.AddCommand(treasureChestScanCmd)
}

// --- types ---

type sqEntry struct {
	ID              string   `yaml:"id"`
	Description     string   `yaml:"description"`
	Strategy        string   `yaml:"strategy"`
	EstimatedImpact string   `yaml:"estimated_impact"`
	Dependencies    []string `yaml:"dependencies"`
	Status          string   `yaml:"status"`
}

type scannedMission struct {
	MissionID  string
	SQs        []sqEntry
	TaskTitles []string
}

type cluster struct {
	ID            string
	CitedMissions []string
	Tags          []string
	GeneratedAt   string
}

type gap struct {
	ID            string
	CitedMissions []string
	Status        string
	Dependencies  []string
	GeneratedAt   string
}

// --- command ---

func runTreasureChestScan(cmd *cobra.Command, _ []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest scan: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRoot, cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest scan: %w", err)
	}
	_, basePath, err := resolveDojoRoots(root)
	if err != nil {
		return fmt.Errorf("treasure-chest scan: %w", err)
	}

	missions, err := scanMissions(basePath)
	if err != nil {
		return fmt.Errorf("treasure-chest scan: %w", err)
	}

	clusters := buildClusters(missions)
	gaps := buildGaps(missions)

	clustersDir := filepath.Join(root, "treasure", "clusters")
	gapsDir := filepath.Join(root, "treasure", "gaps")

	if treasureChestScanDryRun {
		printScanDryRun(missions, clusters, gaps, clustersDir, gapsDir)
		return nil
	}

	if err := writeScanOutputs(clustersDir, clusters, gapsDir, gaps); err != nil {
		return fmt.Errorf("treasure-chest scan: %w", err)
	}

	fmt.Printf("[Strategist] treasure-chest scan: %d mission(s) scanned, %d cluster(s) and %d gap(s) written\n",
		len(missions), len(clusters), len(gaps))
	return nil
}

func printScanDryRun(missions []scannedMission, clusters []cluster, gaps []gap, clustersDir, gapsDir string) {
	fmt.Printf("[Strategist] treasure-chest scan (dry-run): %d mission(s) scanned, would write %d cluster(s), %d gap(s)\n",
		len(missions), len(clusters), len(gaps))
	for _, c := range clusters {
		fmt.Printf("  cluster: %s\n", filepath.Join(clustersDir, c.ID+".md"))
	}
	for _, g := range gaps {
		fmt.Printf("  gap: %s\n", filepath.Join(gapsDir, g.ID+".md"))
	}
}

func writeScanOutputs(clustersDir string, clusters []cluster, gapsDir string, gaps []gap) error {
	if err := regenerateDir(clustersDir); err != nil {
		return err
	}
	for _, c := range clusters {
		if err := writeClusterFile(clustersDir, c); err != nil {
			return err
		}
	}

	if err := regenerateDir(gapsDir); err != nil {
		return err
	}
	for _, g := range gaps {
		if err := writeGapFile(gapsDir, g); err != nil {
			return err
		}
	}
	return nil
}

// --- mission scanning ---

var taskTitleRe = regexp.MustCompile(`^##\s+Task\s+\d+\s*[—-]\s*(.+)$`)

func scanMissions(basePath string) ([]scannedMission, error) {
	var missions []scannedMission

	for _, sub := range []string{"refined", "done"} {
		found, err := scanMissionsInDir(filepath.Join(basePath, sub))
		if err != nil {
			return nil, err
		}
		missions = append(missions, found...)
	}

	sort.Slice(missions, func(i, j int) bool { return missions[i].MissionID < missions[j].MissionID })
	return missions, nil
}

// scanMissionsInDir parses tasks.md for every mission directory under dir. A missing
// dir (base_path not yet populated) is not an error — it just yields no missions.
func scanMissionsInDir(dir string) ([]scannedMission, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	var missions []scannedMission
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		tasksPath := filepath.Join(dir, e.Name(), "tasks.md")
		m, err := parseMissionTasks(e.Name(), tasksPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		missions = append(missions, m)
	}
	return missions, nil
}

// parseMissionTasks extracts task titles and the side_quests_approved: block from a
// tasks.md file. The block is YAML embedded in markdown with backtick-wrapped scalars
// (e.g. “ `SQ-001` “) — backticks are stripped before unmarshaling.
func parseMissionTasks(missionID, path string) (scannedMission, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304
	if err != nil {
		return scannedMission{}, fmt.Errorf("read %s: %w", path, err)
	}
	lines := strings.Split(string(raw), "\n")

	var taskTitles []string
	for _, line := range lines {
		if m := taskTitleRe.FindStringSubmatch(line); m != nil {
			taskTitles = append(taskTitles, strings.TrimSpace(m[1]))
		}
	}

	startIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "side_quests_approved:") {
			startIdx = i
			break
		}
	}

	var sqs []sqEntry
	if startIdx != -1 {
		endIdx := len(lines)
		for i := startIdx + 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "## ") {
				endIdx = i
				break
			}
		}
		block := strings.ReplaceAll(strings.Join(lines[startIdx:endIdx], "\n"), "`", "")
		var parsed struct {
			SideQuestsApproved []sqEntry `yaml:"side_quests_approved"`
		}
		if err := yaml.Unmarshal([]byte(block), &parsed); err != nil {
			return scannedMission{}, fmt.Errorf("parse side_quests_approved in %s: %w", path, err)
		}
		sqs = parsed.SideQuestsApproved
	}

	return scannedMission{MissionID: missionID, SQs: sqs, TaskTitles: taskTitles}, nil
}

// --- cluster pass ---

var (
	tagWordRe    = regexp.MustCompile(`[a-z0-9]+`)
	tagStopwords = map[string]bool{
		"this": true, "that": true, "with": true, "from": true, "have": true,
		"will": true, "into": true, "than": true, "then": true, "when": true,
		"which": true, "their": true, "there": true, "these": true, "those": true,
		"about": true, "after": true, "before": true, "under": true, "over": true,
	}
)

func extractTags(m scannedMission) []string {
	text := strings.ToLower(strings.Join(m.TaskTitles, " "))
	for _, sq := range m.SQs {
		text += " " + strings.ToLower(sq.Description)
	}
	words := tagWordRe.FindAllString(text, -1)
	seen := make(map[string]bool)
	var tags []string
	for _, w := range words {
		if len(w) < 4 || tagStopwords[w] || seen[w] {
			continue
		}
		seen[w] = true
		tags = append(tags, w)
	}
	sort.Strings(tags)
	return tags
}

func sharedTagCount(a, b []string) int {
	set := make(map[string]bool, len(a))
	for _, t := range a {
		set[t] = true
	}
	n := 0
	for _, t := range b {
		if set[t] {
			n++
		}
	}
	return n
}

func buildClusters(missions []scannedMission) []cluster {
	tagsByMission, ids := tagsForMissions(missions)
	uf := unionFindFromSharedTags(ids, tagsByMission)

	var clusters []cluster
	for _, members := range uf.groups() {
		if c, ok := clusterFromGroup(members, tagsByMission); ok {
			clusters = append(clusters, c)
		}
	}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].ID < clusters[j].ID })
	return clusters
}

func tagsForMissions(missions []scannedMission) (map[string][]string, []string) {
	tagsByMission := make(map[string][]string, len(missions))
	ids := make([]string, 0, len(missions))
	for _, m := range missions {
		tagsByMission[m.MissionID] = extractTags(m)
		ids = append(ids, m.MissionID)
	}
	sort.Strings(ids)
	return tagsByMission, ids
}

// unionFindFromSharedTags unions any two missions that share 2+ tags.
func unionFindFromSharedTags(ids []string, tagsByMission map[string][]string) *unionFind {
	uf := newUnionFind(ids)
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if sharedTagCount(tagsByMission[ids[i]], tagsByMission[ids[j]]) >= 2 {
				uf.union(ids[i], ids[j])
			}
		}
	}
	return uf
}

// clusterFromGroup builds a cluster from a union-find group, keeping only groups of
// 2+ missions that share 2+ tags across all members. ok is false when the group does
// not qualify as a cluster.
func clusterFromGroup(members []string, tagsByMission map[string][]string) (c cluster, ok bool) {
	if len(members) < 2 {
		return cluster{}, false
	}
	sort.Strings(members)
	freq := make(map[string]int)
	for _, id := range members {
		for _, tag := range tagsByMission[id] {
			freq[tag]++
		}
	}
	var sharedTags []string
	for tag, count := range freq {
		if count >= 2 {
			sharedTags = append(sharedTags, tag)
		}
	}
	if len(sharedTags) == 0 {
		return cluster{}, false
	}
	sort.Strings(sharedTags)
	return cluster{
		ID:            clusterID(sharedTags),
		CitedMissions: members,
		Tags:          sharedTags,
		GeneratedAt:   nowISO(),
	}, true
}

func clusterID(tags []string) string {
	n := tags
	if len(n) > 2 {
		n = n[:2]
	}
	return "cluster-" + strings.Join(n, "-")
}

// --- gap pass ---

func buildGaps(missions []scannedMission) []gap {
	byID := make(map[string]*gap)
	var order []string
	for _, m := range missions {
		for _, sq := range m.SQs {
			if sq.Status != "sq_pending" {
				continue
			}
			id := gapID(sq.ID)
			if g, ok := byID[id]; ok {
				g.CitedMissions = append(g.CitedMissions, m.MissionID)
				continue
			}
			byID[id] = &gap{
				ID:            id,
				CitedMissions: []string{m.MissionID},
				Status:        sq.Status,
				Dependencies:  sq.Dependencies,
				GeneratedAt:   nowISO(),
			}
			order = append(order, id)
		}
	}
	sort.Strings(order)
	gaps := make([]gap, 0, len(order))
	for _, id := range order {
		gaps = append(gaps, *byID[id])
	}
	return gaps
}

func gapID(sqID string) string {
	return strings.ToLower(sqID)
}

// --- writers ---

func regenerateDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301
		return fmt.Errorf("create %s: %w", dir, err)
	}
	return nil
}

func writeClusterFile(dir string, c cluster) error {
	content := fmt.Sprintf(`---
id: %s
cited_missions: [%s]
tags: [%s]
trust: T2
generated_at: %s
---

# Cluster: %s

Recurring theme across: %s
`, c.ID, strings.Join(c.CitedMissions, ", "), strings.Join(c.Tags, ", "), c.GeneratedAt, c.ID, strings.Join(c.CitedMissions, ", "))
	path := filepath.Join(dir, c.ID+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // G306
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeGapFile(dir string, g gap) error {
	deps := "[]"
	if len(g.Dependencies) > 0 {
		deps = "[" + strings.Join(g.Dependencies, ", ") + "]"
	}
	content := fmt.Sprintf(`---
id: %s
cited_missions: [%s]
status: %s
dependencies: %s
trust: T2
generated_at: %s
---

# Gap: %s

Still pending in: %s
`, g.ID, strings.Join(g.CitedMissions, ", "), g.Status, deps, g.GeneratedAt, g.ID, strings.Join(g.CitedMissions, ", "))
	path := filepath.Join(dir, g.ID+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // G306
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// --- union-find ---

type unionFind struct {
	parent map[string]string
}

func newUnionFind(ids []string) *unionFind {
	p := make(map[string]string, len(ids))
	for _, id := range ids {
		p[id] = id
	}
	return &unionFind{parent: p}
}

func (u *unionFind) find(x string) string {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b string) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[ra] = rb
	}
}

func (u *unionFind) groups() map[string][]string {
	out := make(map[string][]string)
	for id := range u.parent {
		root := u.find(id)
		out[root] = append(out[root], id)
	}
	return out
}
