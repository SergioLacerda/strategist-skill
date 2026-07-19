package treasure

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// MissionHistoryChestID is the synthetic chest used for mined mission history.
const MissionHistoryChestID = "mission-history"

// BuildJewelCandidates converts mined clusters and gaps into proposed jewels.
func BuildJewelCandidates(clusters []Cluster, gaps []Gap) []Jewel {
	return BuildJewelCandidatesWithPolicy(clusters, gaps, DefaultScoringPolicy())
}

// BuildJewelCandidatesWithPolicy converts mined clusters and gaps into proposed jewels.
func BuildJewelCandidatesWithPolicy(clusters []Cluster, gaps []Gap, policy ScoringPolicy) []Jewel {
	candidates := make([]Jewel, 0, len(clusters)+len(gaps))
	for _, c := range clusters {
		candidates = append(candidates, Jewel{
			ID:         "jewel-" + c.ID,
			ChestID:    MissionHistoryChestID,
			Kind:       "pattern",
			Statement:  fmt.Sprintf("Recurring theme across missions: %s", strings.Join(c.Tags, ", ")),
			SourceRefs: missionSourceRefs(c.CitedMissions),
			Trust:      "T2",
			Status:     domain.JewelStatusProposed,
			ReviewedBy: "agent",
			Score: JewelScore{
				Value: ClusterCandidateScoreWithPolicy(c, policy),
				Reasons: []string{
					fmt.Sprintf("recurring across %d missions", len(c.CitedMissions)),
					"shared tags: " + strings.Join(c.Tags, ", "),
				},
			},
			Applicability: JewelApplicability{Scope: []string{"all"}},
		})
	}
	for _, g := range gaps {
		candidates = append(candidates, Jewel{
			ID:         "jewel-gap-" + g.ID,
			ChestID:    MissionHistoryChestID,
			Kind:       "gap",
			Statement:  fmt.Sprintf("Open side quest %s still pending", g.ID),
			SourceRefs: missionSourceRefs(g.CitedMissions),
			Trust:      "T2",
			Status:     domain.JewelStatusProposed,
			ReviewedBy: "agent",
			Score: JewelScore{
				Value:   GapCandidateScoreWithPolicy(g, policy),
				Reasons: []string{fmt.Sprintf("still pending in %d mission(s)", len(g.CitedMissions))},
			},
			Applicability: JewelApplicability{Scope: []string{"all"}},
		})
	}
	return candidates
}

func missionSourceRefs(missionIDs []string) []string {
	refs := make([]string, 0, len(missionIDs))
	for _, id := range missionIDs {
		refs = append(refs, MissionHistoryChestID+"#"+id)
	}
	return refs
}

// ClusterCandidateScore scores a recurring-mission cluster candidate.
func ClusterCandidateScore(c Cluster) int {
	return ClusterCandidateScoreWithPolicy(c, DefaultScoringPolicy())
}

// ClusterCandidateScoreWithPolicy scores a recurring-mission cluster candidate.
func ClusterCandidateScoreWithPolicy(c Cluster, policy ScoringPolicy) int {
	v := policy.ClusterBase + len(c.CitedMissions)*policy.ClusterMissionWeight + len(c.Tags)*policy.ClusterTagWeight
	return clampScore(v, policy.MaxScore)
}

// GapCandidateScore scores an open side-quest gap candidate.
func GapCandidateScore(g Gap) int {
	return GapCandidateScoreWithPolicy(g, DefaultScoringPolicy())
}

// GapCandidateScoreWithPolicy scores an open side-quest gap candidate.
func GapCandidateScoreWithPolicy(g Gap, policy ScoringPolicy) int {
	v := policy.GapBase + len(g.CitedMissions)*policy.GapMissionWeight
	return clampScore(v, policy.MaxScore)
}

func clampScore(value, maxScore int) int {
	if value > maxScore {
		return maxScore
	}
	return value
}

// WriteProposedJewels appends new proposed jewels while skipping existing ids.
func WriteProposedJewels(root string, candidates []Jewel) (written, skipped int, err error) {
	if len(candidates) == 0 {
		return 0, 0, nil
	}

	docs, written, skipped, err := buildProposedJewelDocs(root, candidates)
	if err != nil {
		return written, skipped, err
	}

	if written == 0 {
		return written, skipped, nil
	}
	if err := writeProposedJewelDocs(root, docs); err != nil {
		return written, skipped, err
	}
	return written, skipped, nil
}

func buildProposedJewelDocs(root string, candidates []Jewel) (map[string]*yaml.Node, int, int, error) {
	existing, err := ExistingJewelIDs(root)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("index proposed jewels: %w", err)
	}
	proposedAt := time.Now().UTC().Format("2006-01-02")
	docs := make(map[string]*yaml.Node)
	written, skipped := 0, 0
	for _, cand := range candidates {
		if existing[cand.ID] {
			skipped++
			continue
		}
		if err := appendCandidateToPartition(root, docs, cand, proposedAt); err != nil {
			return docs, written, skipped, err
		}
		existing[cand.ID] = true
		written++
	}
	return docs, written, skipped, nil
}

func writeProposedJewelDocs(root string, docs map[string]*yaml.Node) error {
	if err := os.MkdirAll(filepath.Join(root, "jewels"), 0o755); err != nil {
		return fmt.Errorf("index proposed jewels: create jewels/: %w", err)
	}
	var writes []YAMLWrite
	for path, doc := range docs {
		writes = append(writes, YAMLWrite{Path: path, Doc: doc})
	}
	sort.Slice(writes, func(i, j int) bool { return writes[i].Path < writes[j].Path })
	if _, err := WriteYAMLNodes(writes...); err != nil {
		return fmt.Errorf("index proposed jewels: %w", err)
	}
	return nil
}

func appendCandidateToPartition(root string, docs map[string]*yaml.Node, cand Jewel, proposedAt string) error {
	path := JewelPartitionPath(root, cand.ChestID)
	doc, ok := docs[path]
	if !ok {
		var err error
		doc, err = ReadOrCreateJewelsDocument(path)
		if err != nil {
			return fmt.Errorf("index proposed jewels: %w", err)
		}
		docs[path] = doc
	}
	mapping, err := RootMapping(doc)
	if err != nil {
		return fmt.Errorf("index proposed jewels: %w", err)
	}
	seq := FindOrCreateSequence(mapping, "jewels")
	seq.Style = 0
	var entry yaml.Node
	if err := entry.Encode(cand); err != nil {
		return fmt.Errorf("index proposed jewels: encode %s: %w", cand.ID, err)
	}
	AppendJewelHistory(&entry, domain.JewelStatusProposed, proposedAt, "agent", "")
	seq.Content = append(seq.Content, &entry)
	return nil
}

// ExistingJewelIDs returns all ids present across monolithic and partitioned jewel manifests.
func ExistingJewelIDs(root string) (map[string]bool, error) {
	paths, err := JewelManifestPaths(root)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool)
	for _, path := range paths {
		doc, err := ReadYAMLNode(path)
		if err != nil {
			return nil, err
		}
		if err := collectJewelIDs(doc, ids); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

func collectJewelIDs(doc *yaml.Node, ids map[string]bool) error {
	mapping, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := MappingValue(mapping, "jewels")
	if seq == nil {
		return nil
	}
	for _, entry := range seq.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}
		if id := MappingValue(entry, "id"); id != nil {
			ids[id.Value] = true
		}
	}
	return nil
}
