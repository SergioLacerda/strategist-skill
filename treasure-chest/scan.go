package treasure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// ScanWarning records a mission file skipped by tolerant scan callers.
type ScanWarning struct {
	Path string
	Err  error
}

func (w ScanWarning) Error() string {
	if w.Path == "" {
		return w.Err.Error()
	}
	return fmt.Sprintf("%s: %v", w.Path, w.Err)
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

// ScanResult is the mined-and-persisted output of RunScanPipeline.
type ScanResult struct {
	Missions []ScannedMission
	Clusters []Cluster
	Gaps     []Gap
	Warnings []ScanWarning
}

// RunScanPipeline mines missions under basePath, builds clusters/gaps from
// them, and persists the results under root/treasure/{clusters,gaps}. It is
// the use case behind `treasure-chest index`'s scan phase.
func RunScanPipeline(root, basePath string) (ScanResult, error) {
	missions, warnings, err := ScanMissionsTolerant(basePath)
	if err != nil {
		return ScanResult{}, err
	}
	clusters := BuildClusters(missions)
	gaps := BuildGaps(missions)
	if err := WriteScanOutputs(
		filepath.Join(root, "treasure", "clusters"), clusters,
		filepath.Join(root, "treasure", "gaps"), gaps,
	); err != nil {
		return ScanResult{}, err
	}
	return ScanResult{Missions: missions, Clusters: clusters, Gaps: gaps, Warnings: warnings}, nil
}

// --- mission scanning ---

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

// ScanMissionsTolerant scans refined and done mission directories under basePath,
// skipping inconsistent mission files while reporting them as warnings.
func ScanMissionsTolerant(basePath string) ([]ScannedMission, []ScanWarning, error) {
	var missions []ScannedMission
	var warnings []ScanWarning

	for _, sub := range []string{"refined", "done"} {
		found, foundWarnings, err := ScanMissionsInDirTolerant(filepath.Join(basePath, sub))
		if err != nil {
			return nil, nil, err
		}
		missions = append(missions, found...)
		warnings = append(warnings, foundWarnings...)
	}

	sort.Slice(missions, func(i, j int) bool { return missions[i].MissionID < missions[j].MissionID })
	return missions, warnings, nil
}

// ScanMissionsInDir parses tasks.md for every mission directory under dir. A missing
// dir (base_path not yet populated) is not an error — it just yields no missions.
func ScanMissionsInDir(dir string) ([]ScannedMission, error) {
	entries, err := readMissionDir(dir)
	if err != nil {
		return nil, err
	}

	missions, _, err := scanMissionDirEntries(dir, entries, false)
	return missions, err
}

// ScanMissionsInDirTolerant parses tasks.md files under dir, skipping mission-level
// parse/read failures while preserving directory-level failures as hard errors.
func ScanMissionsInDirTolerant(dir string) ([]ScannedMission, []ScanWarning, error) {
	entries, err := readMissionDir(dir)
	if err != nil {
		return nil, nil, err
	}

	return scanMissionDirEntries(dir, entries, true)
}

func scanMissionDirEntries(dir string, entries []os.DirEntry, tolerant bool) ([]ScannedMission, []ScanWarning, error) {
	if tolerant {
		return scanMissionDirEntriesTolerant(dir, entries)
	}
	missions, err := scanMissionDirEntriesStrict(dir, entries)
	return missions, nil, err
}

func scanMissionDirEntriesStrict(dir string, entries []os.DirEntry) ([]ScannedMission, error) {
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

func scanMissionDirEntriesTolerant(dir string, entries []os.DirEntry) ([]ScannedMission, []ScanWarning, error) {
	var missions []ScannedMission
	var warnings []ScanWarning
	for _, e := range entries {
		m, ok, err := scanMissionDirEntry(dir, e)
		if err != nil {
			warnings = append(warnings, scanMissionWarning(dir, e, err))
			continue
		}
		if ok {
			missions = append(missions, m)
		}
	}
	return missions, warnings, nil
}

func scanMissionWarning(dir string, entry os.DirEntry, err error) ScanWarning {
	return ScanWarning{
		Path: filepath.Join(dir, entry.Name(), "tasks.md"),
		Err:  err,
	}
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
