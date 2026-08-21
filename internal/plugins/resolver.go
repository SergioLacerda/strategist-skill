// Package plugins resolves plugin candidates into deterministic lock graphs.
package plugins

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

const lockSchemaVersion = "strategist-plugin-lock/v1"

// Requirement asks the resolver to materialize one resource by stable ID/kind.
type Requirement struct {
	ID         string
	Kind       string
	Constraint string
}

// Dependency is a candidate's declared dependency edge.
type Dependency struct {
	ID         string
	Kind       string
	Constraint string
	Optional   bool
	Reason     string
}

// Candidate is one local or embedded resource available to the resolver.
type Candidate struct {
	ID           string
	Kind         string
	Version      string
	Digest       string
	Dependencies []Dependency
}

type resolver struct {
	candidates  map[string][]Candidate
	resolved    map[string]Candidate
	constraints map[string][]string
	visiting    map[string]bool
}

// Resolve creates a deterministic digest-pinned lock from embedded/local candidates.
func Resolve(requirements []Requirement, candidates []Candidate) (domain.PluginLock, error) {
	r := newResolver(candidates)
	sortedRequirements := append([]Requirement(nil), requirements...)
	sort.Slice(sortedRequirements, func(i, j int) bool {
		return requirementKey(sortedRequirements[i]) < requirementKey(sortedRequirements[j])
	})
	for _, req := range sortedRequirements {
		if err := r.resolveRequirement(resolveRequest{Requirement: req}, nil); err != nil {
			return domain.PluginLock{}, err
		}
	}
	nodes := lockNodesFromResolved(r.resolved)
	graphDigest := DigestLockNodes(nodes)
	return domain.PluginLock{
		SchemaVersion: lockSchemaVersion,
		ResolutionID:  graphDigest,
		GraphDigest:   graphDigest,
		Nodes:         nodes,
	}, nil
}

// VerifyLock verifies that a lock can replay from local candidates without selection.
func VerifyLock(lock domain.PluginLock, candidates []Candidate) error {
	if lock.SchemaVersion != lockSchemaVersion {
		return fmt.Errorf("lock_schema_unsupported: %s", lock.SchemaVersion)
	}
	if got := DigestLockNodes(lock.Nodes); got != lock.GraphDigest {
		return fmt.Errorf("lock_graph_digest_mismatch: got %s want %s", got, lock.GraphDigest)
	}
	byPinnedIdentity := map[string]bool{}
	for _, candidate := range candidates {
		byPinnedIdentity[candidateKey(candidate)+"@"+candidate.Digest] = true
	}
	for _, node := range lock.Nodes {
		if !byPinnedIdentity[nodeKey(node)+"@"+node.Digest] {
			return fmt.Errorf("lock_replay_digest_missing: %s %s", nodeKey(node), node.Digest)
		}
	}
	return nil
}

// DigestLockNodes returns the canonical digest for a sorted lock graph.
func DigestLockNodes(nodes []domain.PluginLockNode) string {
	canonical := append([]domain.PluginLockNode(nil), nodes...)
	sort.Slice(canonical, func(i, j int) bool {
		return nodeKey(canonical[i]) < nodeKey(canonical[j])
	})
	var b strings.Builder
	for _, node := range canonical {
		b.WriteString(node.Kind)
		b.WriteString("\t")
		b.WriteString(node.ID)
		b.WriteString("\t")
		b.WriteString(node.Digest)
		b.WriteString("\n")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("sha256:%x", sum)
}

func newResolver(candidates []Candidate) *resolver {
	index := map[string][]Candidate{}
	for _, candidate := range candidates {
		index[candidateKey(candidate)] = append(index[candidateKey(candidate)], candidate)
	}
	for key := range index {
		sort.Slice(index[key], func(i, j int) bool {
			return compareCandidate(index[key][i], index[key][j]) < 0
		})
	}
	return &resolver{
		candidates:  index,
		resolved:    map[string]Candidate{},
		constraints: map[string][]string{},
		visiting:    map[string]bool{},
	}
}

type resolveRequest struct {
	Requirement
	Optional bool
}

func (r *resolver) resolveRequirement(req resolveRequest, path []string) error {
	key := requirementKey(req.Requirement)
	r.constraints[key] = append(r.constraints[key], normalizeConstraint(req.Constraint))
	if r.visiting[key] {
		return fmt.Errorf("dependency_cycle: %s", strings.Join(append(path, key), " -> "))
	}
	if resolved, ok := r.resolved[key]; ok {
		if !candidateSatisfiesAll(resolved, r.constraints[key]) {
			return conflictError(key, r.constraints[key])
		}
		return nil
	}
	r.visiting[key] = true
	defer delete(r.visiting, key)

	selected, ok := r.selectCandidate(req.Requirement)
	if !ok {
		if req.Optional {
			return nil
		}
		return fmt.Errorf("dependency_missing: %s %s", key, req.Constraint)
	}
	r.resolved[key] = selected

	for _, dep := range sortedDependencies(selected.Dependencies) {
		depReq := resolveRequest{Requirement: Requirement{ID: dep.ID, Kind: dep.Kind, Constraint: dep.Constraint}, Optional: dep.Optional}
		if err := r.resolveRequirement(depReq, append(path, key)); err != nil {
			return err
		}
	}
	if !candidateSatisfiesAll(selected, r.constraints[key]) {
		return conflictError(key, r.constraints[key])
	}
	return nil
}

func (r *resolver) selectCandidate(req Requirement) (Candidate, bool) {
	key := requirementKey(req)
	constraints := r.constraints[key]
	for _, candidate := range r.candidates[key] {
		if candidateSatisfiesAll(candidate, constraints) {
			return candidate, true
		}
	}
	return Candidate{}, false
}

func lockNodesFromResolved(resolved map[string]Candidate) []domain.PluginLockNode {
	nodes := make([]domain.PluginLockNode, 0, len(resolved))
	for _, candidate := range resolved {
		nodes = append(nodes, domain.PluginLockNode{
			ID:     candidate.ID,
			Kind:   candidate.Kind,
			Digest: candidate.Digest,
		})
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodeKey(nodes[i]) < nodeKey(nodes[j])
	})
	return nodes
}
