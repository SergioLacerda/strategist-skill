package treasure

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// --- SQ-010 (Track T-F): treasure-chest scan ---
//
// Mines .analysis/refined/**/tasks.md and .analysis/done/** (never pending/ or archived/
// reports) for recurring clusters (2+ missions sharing 2+ tags) and open gaps (side quests
// still status: sq_pending), per .analysis/refined/bau-tesouro-sq003-004-007/design.md and
// .analysis/refined/bau-tesouro-sq010-scan-runtime/design.md. Lexical/tag matching only — no
// embeddings, no semantic retrieval.

// SQEntry is a side_quests_approved entry parsed from a mission tasks.md file.
type SQEntry struct {
	ID              string   `yaml:"id"`
	Description     string   `yaml:"description"`
	Strategy        string   `yaml:"strategy"`
	EstimatedImpact string   `yaml:"estimated_impact"`
	Dependencies    []string `yaml:"dependencies"`
	Status          string   `yaml:"status"`
}

// ScannedMission is the mined task and side-quest data for one mission.
type ScannedMission struct {
	MissionID  string
	SQs        []SQEntry
	TaskTitles []string
}

// Cluster is a recurring theme shared across mined missions.
type Cluster struct {
	ID            string
	CitedMissions []string
	Tags          []string
	GeneratedAt   string
}

// Gap is an open side quest still pending in mined mission history.
type Gap struct {
	ID            string
	CitedMissions []string
	Status        string
	Dependencies  []string
	GeneratedAt   string
}

// WriteScanOutputs regenerates cluster and gap output directories from scan results.
func WriteScanOutputs(clustersDir string, clusters []Cluster, gapsDir string, gaps []Gap) error {
	if err := RegenerateDir(clustersDir); err != nil {
		return err
	}
	if err := writeClusterFiles(clustersDir, clusters); err != nil {
		return err
	}
	if err := RegenerateDir(gapsDir); err != nil {
		return err
	}
	return writeGapFiles(gapsDir, gaps)
}

func writeClusterFiles(clustersDir string, clusters []Cluster) error {
	for _, c := range clusters {
		if err := WriteClusterFile(clustersDir, c); err != nil {
			return err
		}
	}
	return nil
}

func writeGapFiles(gapsDir string, gaps []Gap) error {
	for _, g := range gaps {
		if err := WriteGapFile(gapsDir, g); err != nil {
			return err
		}
	}
	return nil
}

// --- mission scanning ---

var taskTitleRe = regexp.MustCompile(`^##\s+Task\s+\d+\s*[—-]\s*(.+)$`)

// ScanMissions scans refined and done mission directories under basePath.
func ScanMissions(basePath string) ([]ScannedMission, error) {
	var missions []ScannedMission

	for _, sub := range []string{"refined", "done"} {
		found, err := ScanMissionsInDir(filepath.Join(basePath, sub))
		if err != nil {
			return nil, err
		}
		missions = append(missions, found...)
	}

	sort.Slice(missions, func(i, j int) bool { return missions[i].MissionID < missions[j].MissionID })
	return missions, nil
}

// ScanMissionsInDir parses tasks.md for every mission directory under dir. A missing
// dir (base_path not yet populated) is not an error — it just yields no missions.
func ScanMissionsInDir(dir string) ([]ScannedMission, error) {
	entries, err := readMissionDir(dir)
	if err != nil {
		return nil, err
	}

	var missions []ScannedMission
	for _, e := range entries {
		m, ok, err := scanMissionDirEntry(dir, e)
		if err != nil {
			return nil, err
		}
		if ok {
			missions = append(missions, m)
		}
	}
	return missions, nil
}

func readMissionDir(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err == nil {
		return entries, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, fmt.Errorf("read %s: %w", dir, err)
}

func scanMissionDirEntry(dir string, entry os.DirEntry) (ScannedMission, bool, error) {
	if !entry.IsDir() {
		return ScannedMission{}, false, nil
	}
	tasksPath := filepath.Join(dir, entry.Name(), "tasks.md")
	m, err := ParseMissionTasks(entry.Name(), tasksPath)
	if err == nil {
		return m, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return ScannedMission{}, false, nil
	}
	return ScannedMission{}, false, err
}

// ParseMissionTasks extracts task titles and the side_quests_approved: block from a
// tasks.md file. The block is YAML embedded in markdown with optional fenced YAML
// and backtick-wrapped scalars (e.g. “ `SQ-001` “) — backticks are stripped before
// unmarshaling.
func ParseMissionTasks(missionID, path string) (ScannedMission, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304
	if err != nil {
		return ScannedMission{}, fmt.Errorf("read %s: %w", path, err)
	}
	lines := strings.Split(string(raw), "\n")

	sqs, err := parseMissionSideQuests(lines, path)
	if err != nil {
		return ScannedMission{}, err
	}
	return ScannedMission{MissionID: missionID, SQs: sqs, TaskTitles: parseMissionTaskTitles(lines)}, nil
}

func parseMissionTaskTitles(lines []string) []string {
	var taskTitles []string
	for _, line := range lines {
		if m := taskTitleRe.FindStringSubmatch(line); m != nil {
			taskTitles = append(taskTitles, strings.TrimSpace(m[1]))
		}
	}
	return taskTitles
}

func parseMissionSideQuests(lines []string, path string) ([]SQEntry, error) {
	startIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "side_quests_approved:") {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return nil, nil
	}
	return parseSideQuestBlock(lines[startIdx:sideQuestBlockEnd(lines, startIdx)], path)
}

func sideQuestBlockEnd(lines []string, startIdx int) int {
	for i := startIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			return i
		}
	}
	return len(lines)
}

func parseSideQuestBlock(lines []string, path string) ([]SQEntry, error) {
	block := normalizeSideQuestYAML(lines)
	block = normalizeLegacySideQuestFields(block)
	block = strings.ReplaceAll(block, "`", "")
	var parsed struct {
		SideQuestsApproved []SQEntry `yaml:"side_quests_approved"`
	}
	if err := yaml.Unmarshal([]byte(block), &parsed); err != nil {
		return nil, fmt.Errorf("parse side_quests_approved in %s: %w", path, err)
	}
	return parsed.SideQuestsApproved, nil
}

func normalizeSideQuestYAML(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	contentStart := firstSideQuestContentLine(lines)
	if contentStart >= len(lines) || !isFenceLine(lines[contentStart]) {
		return strings.Join(lines, "\n")
	}
	return lines[0] + "\n" + strings.Join(fencedBody(lines, contentStart), "\n")
}

func firstSideQuestContentLine(lines []string) int {
	contentStart := 1
	for contentStart < len(lines) && strings.TrimSpace(lines[contentStart]) == "" {
		contentStart++
	}
	return contentStart
}

func fencedBody(lines []string, contentStart int) []string {
	var body []string
	for i := contentStart + 1; i < len(lines); i++ {
		if isFenceLine(lines[i]) {
			break
		}
		body = append(body, lines[i])
	}
	return body
}

func isFenceLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "```") {
		return false
	}
	lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
	return lang == "" || lang == "yaml" || lang == "yml"
}

func normalizeLegacySideQuestFields(block string) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent > 0 && strings.HasPrefix(trimmed, "- ") && looksLikeMappingField(strings.TrimPrefix(trimmed, "- ")) {
			trimmed = strings.TrimPrefix(trimmed, "- ")
			lines[i] = line[:indent] + trimmed
		}
		if isNoneDependencyField(trimmed) {
			lines[i] = line[:indent] + "dependencies: []"
		}
	}
	return strings.Join(lines, "\n")
}

func looksLikeMappingField(value string) bool {
	key, _, ok := strings.Cut(value, ":")
	if !ok || key == "" {
		return false
	}
	for _, r := range key {
		if !isMappingKeyRune(r) {
			return false
		}
	}
	return true
}

func isMappingKeyRune(r rune) bool {
	return isASCIILetter(r) || (r >= '0' && r <= '9') || r == '_' || r == '-'
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isNoneDependencyField(trimmed string) bool {
	key, value, ok := strings.Cut(trimmed, ":")
	if !ok || strings.TrimSpace(key) != "dependencies" {
		return false
	}
	value = strings.ToLower(strings.Trim(strings.TrimSpace(value), "`\"'"))
	return value == "none" || value == "null" || value == "n/a"
}

// --- Cluster pass ---

var (
	tagWordRe    = regexp.MustCompile(`[a-z0-9]+`)
	tagStopwords = map[string]bool{
		"this": true, "that": true, "with": true, "from": true, "have": true,
		"will": true, "into": true, "than": true, "then": true, "when": true,
		"which": true, "their": true, "there": true, "these": true, "those": true,
		"about": true, "after": true, "before": true, "under": true, "over": true,
	}
)

// ExtractTags derives normalized lexical tags from mission task titles and side quests.
func ExtractTags(m ScannedMission) []string {
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

// BuildClusters groups missions that share enough lexical tags into recurring clusters.
func BuildClusters(missions []ScannedMission) []Cluster {
	tagsByMission, ids := tagsForMissions(missions)
	uf := unionFindFromSharedTags(ids, tagsByMission)

	var clusters []Cluster
	for _, members := range uf.groups() {
		if c, ok := clusterFromGroup(members, tagsByMission); ok {
			clusters = append(clusters, c)
		}
	}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].ID < clusters[j].ID })
	return clusters
}

func tagsForMissions(missions []ScannedMission) (map[string][]string, []string) {
	tagsByMission := make(map[string][]string, len(missions))
	ids := make([]string, 0, len(missions))
	for _, m := range missions {
		tagsByMission[m.MissionID] = ExtractTags(m)
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

// clusterFromGroup builds a Cluster from a union-find group, keeping only groups of
// 2+ missions that share 2+ tags across all members. ok is false when the group does
// not qualify as a Cluster.
func clusterFromGroup(members []string, tagsByMission map[string][]string) (c Cluster, ok bool) {
	if len(members) < 2 {
		return Cluster{}, false
	}
	sort.Strings(members)
	sharedTags := sharedTagsForMembers(members, tagsByMission)
	if len(sharedTags) == 0 {
		return Cluster{}, false
	}
	sort.Strings(sharedTags)
	return Cluster{
		ID:            ClusterID(sharedTags),
		CitedMissions: members,
		Tags:          sharedTags,
		GeneratedAt:   nowISO(),
	}, true
}

func sharedTagsForMembers(members []string, tagsByMission map[string][]string) []string {
	freq := tagFrequency(members, tagsByMission)
	var sharedTags []string
	for tag, count := range freq {
		if count >= 2 {
			sharedTags = append(sharedTags, tag)
		}
	}
	sort.Strings(sharedTags)
	return sharedTags
}

func tagFrequency(members []string, tagsByMission map[string][]string) map[string]int {
	freq := make(map[string]int)
	for _, id := range members {
		for _, tag := range tagsByMission[id] {
			freq[tag]++
		}
	}
	return freq
}

// ClusterID builds the stable id for a cluster from its tags.
func ClusterID(tags []string) string {
	n := tags
	if len(n) > 2 {
		n = n[:2]
	}
	return "cluster-" + strings.Join(n, "-")
}

// --- Gap pass ---

// BuildGaps returns still-pending side quests across scanned missions.
func BuildGaps(missions []ScannedMission) []Gap {
	byID := make(map[string]*Gap)
	var order []string
	for _, m := range missions {
		for _, sq := range m.SQs {
			order = appendPendingGap(byID, order, m.MissionID, sq)
		}
	}
	return sortedGaps(byID, order)
}

func appendPendingGap(byID map[string]*Gap, order []string, missionID string, sq SQEntry) []string {
	if sq.Status != "sq_pending" {
		return order
	}
	id := GapID(sq.ID)
	if g, ok := byID[id]; ok {
		g.CitedMissions = append(g.CitedMissions, missionID)
		return order
	}
	byID[id] = &Gap{
		ID:            id,
		CitedMissions: []string{missionID},
		Status:        sq.Status,
		Dependencies:  sq.Dependencies,
		GeneratedAt:   nowISO(),
	}
	return append(order, id)
}

func sortedGaps(byID map[string]*Gap, order []string) []Gap {
	sort.Strings(order)
	gaps := make([]Gap, 0, len(order))
	for _, id := range order {
		gaps = append(gaps, *byID[id])
	}
	return gaps
}

// GapID normalizes a side-quest id for gap artifact filenames and ids.
func GapID(sqID string) string {
	return strings.ToLower(sqID)
}

// --- writers ---

// RegenerateDir recreates a generated output directory from scratch.
func RegenerateDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301
		return fmt.Errorf("create %s: %w", dir, err)
	}
	return nil
}

// WriteClusterFile writes one cluster artifact.
func WriteClusterFile(dir string, c Cluster) error {
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

// WriteGapFile writes one gap artifact.
func WriteGapFile(dir string, g Gap) error {
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
