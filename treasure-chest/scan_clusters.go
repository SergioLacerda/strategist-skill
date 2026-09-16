package treasure

import (
	"regexp"
	"sort"
	"strings"
)

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
