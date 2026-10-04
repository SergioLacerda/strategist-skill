package application

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// MaterializeMissionContext validates that the mission exists, then delegates
// bounded context materialization to the domain through its reader port.
// Filesystem ownership stays with the caller-provided reader.
func MaterializeMissionContext(
	load func(root, missionID string) (*domain.MissionEngine, domain.MissionEngineStatus, error),
	root, missionID string,
	reader domain.ContextReader,
	references []domain.ContextReference,
	maxRefs, maxBytes int,
) (domain.MaterializedContext, error) {
	if load == nil {
		return domain.MaterializedContext{}, fmt.Errorf("mission context: mission loader is required")
	}
	if _, _, err := load(root, missionID); err != nil {
		return domain.MaterializedContext{}, err
	}
	materialized, err := domain.MaterializeContext(reader, references, maxRefs, maxBytes)
	if err != nil {
		return domain.MaterializedContext{}, fmt.Errorf("materialize mission context: %w", err)
	}
	return materialized, nil
}
