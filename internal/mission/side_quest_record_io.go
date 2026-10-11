package mission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func sideQuestRecordPath(strategistRoot, missionID, sideQuestID string) (string, error) {
	if strategistRoot == "" {
		return "", errors.New("side_quest_record_invalid: runtime root is required")
	}
	if missionID == "" || strings.ContainsAny(missionID, `/\`) || missionID == "." || missionID == ".." || sideQuestID == "" || strings.ContainsAny(sideQuestID, `/\`) {
		return "", fmt.Errorf("side_quest_record_invalid: malformed mission id %q", missionID)
	}
	name := missionID + ".json"
	if sideQuestID != OAADRSideQuestID(missionID) {
		name = missionID + "-" + strings.ToLower(strings.TrimPrefix(sideQuestID, "OA-")) + ".json"
	}
	return filepath.Join(strategistRoot, "missions", "side-quests", name), nil
}

// LoadAcceptedSideQuest reads and verifies the mission's record. A mission
// with no record reports found=false.
func LoadAcceptedSideQuest(strategistRoot, missionID string) (rec AcceptedSideQuest, found bool, err error) {
	return LoadSideQuest(strategistRoot, missionID, OAADRSideQuestID(missionID))
}

// LoadSideQuest loads one independent opportunity decision.
func LoadSideQuest(strategistRoot, missionID, sideQuestID string) (rec AcceptedSideQuest, found bool, err error) {
	path, err := sideQuestRecordPath(strategistRoot, missionID, sideQuestID)
	if err != nil {
		return AcceptedSideQuest{}, false, err
	}
	raw, err := os.ReadFile(path) //nolint:gosec // path is <root>/missions/side-quests/<validated mission id>.json
	if errors.Is(err, os.ErrNotExist) {
		return AcceptedSideQuest{}, false, nil
	}
	if err != nil {
		return AcceptedSideQuest{}, false, fmt.Errorf("side_quest_record_unreadable: %w", err)
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		return AcceptedSideQuest{}, false, fmt.Errorf("side_quest_record_tampered: parse record: %w", err)
	}
	if err := rec.VerifyIntegrity(); err != nil {
		return AcceptedSideQuest{}, false, err
	}
	if rec.MissionID != missionID {
		return AcceptedSideQuest{}, false, fmt.Errorf("side_quest_record_tampered: record names mission %q, expected %q", rec.MissionID, missionID)
	}
	return rec, true, nil
}

// LoadMissionSideQuests enumerates all durable side-quest intents for a
// mission. Resolution callers can dispatch by record kind without maintaining
// a fixed list of opportunity identifiers.
func LoadMissionSideQuests(strategistRoot, missionID string) ([]AcceptedSideQuest, error) {
	dir := filepath.Join(strategistRoot, "missions", "side-quests")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("side_quest_record_unreadable: enumerate records: %w", err)
	}
	records := make([]AcceptedSideQuest, 0, len(entries))
	for _, entry := range entries {
		record, include, recordErr := loadMissionSideQuestRecord(dir, missionID, entry)
		if recordErr != nil {
			return nil, recordErr
		}
		if !include {
			continue
		}
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool { return records[i].SideQuestID < records[j].SideQuestID })
	return records, nil
}

func loadMissionSideQuestRecord(dir, missionID string, entry os.DirEntry) (AcceptedSideQuest, bool, error) {
	if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
		return AcceptedSideQuest{}, false, nil
	}
	raw, err := os.ReadFile(filepath.Join(dir, entry.Name())) //nolint:gosec // runtime-owned side-quest directory
	if err != nil {
		return AcceptedSideQuest{}, false, fmt.Errorf("side_quest_record_unreadable: %w", err)
	}
	var record AcceptedSideQuest
	if err := json.Unmarshal(raw, &record); err != nil {
		return AcceptedSideQuest{}, false, fmt.Errorf("side_quest_record_tampered: parse record: %w", err)
	}
	if record.MissionID != missionID {
		return AcceptedSideQuest{}, false, nil
	}
	if err := record.VerifyIntegrity(); err != nil {
		return AcceptedSideQuest{}, false, err
	}
	return record, true, nil
}

// saveAcceptedSideQuest seals the digest and writes the record atomically.
// The caller must hold the mission lock.
func saveAcceptedSideQuest(strategistRoot string, rec AcceptedSideQuest) (AcceptedSideQuest, error) {
	path, err := sideQuestRecordPath(strategistRoot, rec.MissionID, rec.SideQuestID)
	if err != nil {
		return AcceptedSideQuest{}, err
	}
	rec, err = rec.sealIntegrity()
	if err != nil {
		return AcceptedSideQuest{}, err
	}
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return AcceptedSideQuest{}, fmt.Errorf("side_quest_record_invalid: encode record: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return AcceptedSideQuest{}, fmt.Errorf("side_quest_record_unwritable: %w", err)
	}
	if err := WriteFileAtomic(path, append(raw, '\n'), 0o600); err != nil {
		return AcceptedSideQuest{}, fmt.Errorf("side_quest_record_unwritable: %w", err)
	}
	return rec, nil
}

func nowStamp(now time.Time) string { return now.UTC().Format(time.RFC3339Nano) }
