package jeweler

import (
	"context"
	"fmt"
	"strings"
)

// RunbookSelectIntent is the host-neutral capability requested when Ranger or
// Archivist needs a runbook/playbook projection from the admitted provider.
const RunbookSelectIntent = "runbook.select"

// RunbookSelectionRequest keeps mission routing and provider access explicit.
// Selection policy remains host-owned; the provider supplies only the bounded
// runbook/playbook candidates and their source evidence.
type RunbookSelectionRequest struct {
	QueryRequest
	Signals []string `json:"signals"`
}

// Validate checks the host-owned runbook selection request.
func (r RunbookSelectionRequest) Validate() error {
	if err := r.QueryRequest.Validate(); err != nil {
		return err
	}
	if len(r.Signals) == 0 {
		return fmt.Errorf("jeweler: runbook.select requires at least one signal")
	}
	for _, signal := range r.Signals {
		if strings.TrimSpace(signal) == "" {
			return fmt.Errorf("jeweler: runbook.select signals must not be empty")
		}
	}
	return nil
}

// SelectRunbooks routes runbook selection through JEWELER. The external
// Treasure Chest provider stores a runbook index entry as a Potion, so this
// capability-specific adapter normalizes potion items to runbooks and filters
// unrelated Jewels from the provider's shared index. Untyped item kinds remain
// invalid and cannot be interpreted as a successful runbook selection.
func (b Broker) SelectRunbooks(ctx context.Context, req RunbookSelectionRequest) (Projection, error) {
	if err := req.Validate(); err != nil {
		return Projection{}, err
	}
	query := req.QueryRequest
	query.Intent = RunbookSelectIntent
	query.Concepts = append(append([]string(nil), query.Concepts...), req.Signals...)
	projection, err := b.Query(ctx, query)
	if err != nil {
		return projection, err
	}
	items, sawKnowledgeOnly, err := projectRunbookItems(projection.Items)
	if err != nil {
		return Projection{}, err
	}
	if len(items) == 0 && sawKnowledgeOnly {
		return Projection{}, fmt.Errorf("jeweler: runbook.select provider returned unsupported item kind %q", "jewel")
	}
	projection.Items = items
	return projection, nil
}

func projectRunbookItems(providerItems []Item) ([]Item, bool, error) {
	items := make([]Item, 0, len(providerItems))
	sawKnowledgeOnly := false
	for _, item := range providerItems {
		projected, knowledgeOnly, err := projectRunbookItem(item)
		if err != nil {
			return nil, false, err
		}
		if knowledgeOnly {
			sawKnowledgeOnly = true
			continue
		}
		items = append(items, projected)
	}
	return items, sawKnowledgeOnly, nil
}

func projectRunbookItem(item Item) (Item, bool, error) {
	switch item.Kind {
	case "potion":
		if strings.TrimSpace(item.Source.Path) == "" {
			return Item{}, false, fmt.Errorf("jeweler: runbook.select potion %q has no source path", item.ID)
		}
		item.Kind = "runbook"
	case "runbook", "playbook":
	case "jewel":
		return Item{}, true, nil
	default:
		return Item{}, false, fmt.Errorf("jeweler: runbook.select provider returned unsupported item kind %q", item.Kind)
	}
	return item, false, nil
}
