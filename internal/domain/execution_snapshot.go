package domain

import (
	"fmt"
	"strings"
)

// Execution artifact roles and snapshot completion policy.
const (
	ExecutionArtifactRolePrimary    = "primary"
	ExecutionArtifactRoleSidecar    = "sidecar"
	ExecutionArtifactRoleSupporting = "supporting"
	ExecutionCompletionAllRequired  = "all_required"
	ExecutionSnapshotVersion        = 1
)

// ResolverKey identifies an execution protocol by domain identity. It is
// deliberately independent of the Weapon that produced an accepted intent.
type ResolverKey struct {
	Kind   string `json:"kind"`
	Origin string `json:"origin"`
}

// Validate checks that the resolver key is complete.
func (k ResolverKey) Validate() error {
	if strings.TrimSpace(k.Kind) == "" || strings.TrimSpace(k.Origin) == "" {
		return fmt.Errorf("execution_resolver_key_invalid: kind and origin are required")
	}
	return nil
}

// ExecutionArtifact is one declared filesystem output in an execution group.
type ExecutionArtifact struct {
	ID       string            `json:"id"`
	Role     string            `json:"role"`
	Path     string            `json:"path"`
	Kind     string            `json:"kind"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Validate checks the artifact identity, role, and project-relative path.
func (a ExecutionArtifact) Validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return fmt.Errorf("execution_artifact_invalid: id is required")
	}
	if err := ValidateProjectRelativePath(a.Path, "execution_artifact.path"); err != nil {
		return err
	}
	if strings.TrimSpace(a.Kind) == "" {
		return fmt.Errorf("execution_artifact_invalid: kind is required")
	}
	switch a.Role {
	case ExecutionArtifactRolePrimary, ExecutionArtifactRoleSidecar, ExecutionArtifactRoleSupporting:
	default:
		return fmt.Errorf("execution_artifact_invalid: unsupported role %q", a.Role)
	}
	return nil
}

// ExecutionGroup is an atomic set of related outputs. All artifacts are
// required for the group to complete.
type ExecutionGroup struct {
	ID               string              `json:"id"`
	Resolver         ResolverKey         `json:"resolver"`
	SideQuestID      string              `json:"side_quest_id,omitempty"`
	Kind             string              `json:"kind"`
	Origin           string              `json:"origin"`
	CompletionPolicy string              `json:"completion_policy"`
	Artifacts        []ExecutionArtifact `json:"artifacts"`
}

// Validate checks group identity, completion policy, and artifact uniqueness.
func (g ExecutionGroup) Validate() error {
	if err := validateExecutionGroupIdentity(g); err != nil {
		return err
	}
	if err := validateExecutionArtifacts(g.Artifacts); err != nil {
		return err
	}
	return validateExecutionGroupSideQuest(g)
}

func validateExecutionGroupIdentity(g ExecutionGroup) error {
	if strings.TrimSpace(g.ID) == "" {
		return fmt.Errorf("execution_group_invalid: id is required")
	}
	if err := g.Resolver.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(g.Kind) == "" || strings.TrimSpace(g.Origin) == "" {
		return fmt.Errorf("execution_group_invalid: kind and origin are required")
	}
	if g.CompletionPolicy != ExecutionCompletionAllRequired {
		return fmt.Errorf("execution_group_invalid: unsupported completion policy %q", g.CompletionPolicy)
	}
	if len(g.Artifacts) == 0 {
		return fmt.Errorf("execution_group_invalid: at least one artifact is required")
	}
	return nil
}

func validateExecutionArtifacts(artifacts []ExecutionArtifact) error {
	seen := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		if _, exists := seen[artifact.ID]; exists {
			return fmt.Errorf("execution_group_invalid: duplicate artifact id %q", artifact.ID)
		}
		if err := artifact.Validate(); err != nil {
			return err
		}
		seen[artifact.ID] = struct{}{}
	}
	return nil
}

func validateExecutionGroupSideQuest(g ExecutionGroup) error {
	if g.Origin == ExecutionTargetOriginAcceptedSideQuest && strings.TrimSpace(g.SideQuestID) == "" {
		return fmt.Errorf("execution_group_invalid: side_quest_id is required for accepted side-quest groups")
	}
	return nil
}

// ExecutionSnapshot is the immutable execution authorization captured at the
// execution boundary.
type ExecutionSnapshot struct {
	Version   int              `json:"version"`
	MissionID string           `json:"mission_id"`
	Groups    []ExecutionGroup `json:"groups"`
}

// Validate checks snapshot version, mission identity, and group uniqueness.
func (s ExecutionSnapshot) Validate() error {
	if s.Version != ExecutionSnapshotVersion {
		return fmt.Errorf("execution_snapshot_invalid: unsupported version %d", s.Version)
	}
	if strings.TrimSpace(s.MissionID) == "" {
		return fmt.Errorf("execution_snapshot_invalid: mission_id is required")
	}
	if len(s.Groups) == 0 {
		return fmt.Errorf("execution_snapshot_invalid: at least one group is required")
	}
	return validateExecutionGroups(s.Groups)
}

func validateExecutionGroups(groups []ExecutionGroup) error {
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if _, exists := seen[group.ID]; exists {
			return fmt.Errorf("execution_snapshot_invalid: duplicate group id %q", group.ID)
		}
		if err := group.Validate(); err != nil {
			return err
		}
		seen[group.ID] = struct{}{}
	}
	return nil
}

// ExecutionResolution is the normalized output of one or more resolvers.
type ExecutionResolution struct {
	Groups []ExecutionGroup `json:"groups"`
}

// Validate checks resolver output using the supplied mission identity.
func (r ExecutionResolution) Validate(missionID string) error {
	snapshot := ExecutionSnapshot{Version: ExecutionSnapshotVersion, MissionID: missionID, Groups: r.Groups}
	return snapshot.Validate()
}
