package domain

import (
	"fmt"
	"sort"
	"strings"
)

// WeaponCompositionOrderedDAG is the only supported composition strategy.
const WeaponCompositionOrderedDAG = "ordered_dag"

// WeaponComponent identifies one component in a Composite Weapon.
type WeaponComponent struct {
	ID        string   `yaml:"id"`
	Required  bool     `yaml:"required"`
	DependsOn []string `yaml:"depends_on,omitempty"`
}

// WeaponComposition is the deterministic dependency graph of a Composite Weapon.
type WeaponComposition struct {
	Strategy   string            `yaml:"strategy"`
	Components []WeaponComponent `yaml:"components"`
}

// Validate checks the strategy, component identities, dependency references,
// and acyclicity of the composition graph.
func (c WeaponComposition) Validate() error {
	if c.Strategy == "" {
		return fmt.Errorf("weapon composition strategy is required")
	}
	if c.Strategy != WeaponCompositionOrderedDAG {
		return fmt.Errorf("weapon composition strategy %q is invalid", c.Strategy)
	}
	if len(c.Components) == 0 {
		return fmt.Errorf("weapon composition components must not be empty")
	}
	ids, err := c.componentIDs()
	if err != nil {
		return err
	}
	if err := c.validateDependencies(ids); err != nil {
		return err
	}
	_, err = c.TopologicalOrder()
	return err
}

func (c WeaponComposition) componentIDs() (map[string]struct{}, error) {
	ids := make(map[string]struct{}, len(c.Components))
	for _, component := range c.Components {
		if strings.TrimSpace(component.ID) == "" {
			return nil, fmt.Errorf("weapon composition component id is required")
		}
		if _, exists := ids[component.ID]; exists {
			return nil, fmt.Errorf("weapon composition contains duplicate component %q", component.ID)
		}
		ids[component.ID] = struct{}{}
	}
	return ids, nil
}

func (c WeaponComposition) validateDependencies(ids map[string]struct{}) error {
	for _, component := range c.Components {
		if err := validateComponentDependencies(component, ids); err != nil {
			return err
		}
	}
	return nil
}

func validateComponentDependencies(component WeaponComponent, ids map[string]struct{}) error {
	for _, dependency := range component.DependsOn {
		if _, exists := ids[dependency]; !exists {
			return fmt.Errorf("weapon composition component %q has unknown component dependency %q", component.ID, dependency)
		}
		if dependency == component.ID {
			return fmt.Errorf("weapon composition contains cycle at component %q", component.ID)
		}
	}
	return nil
}

// TopologicalOrder returns a stable order: dependencies first, lexical order
// for components that are ready at the same time.
func (c WeaponComposition) TopologicalOrder() ([]string, error) {
	indegree, dependents := c.graph()
	order := kahnOrder(indegree, dependents)
	if len(order) != len(indegree) {
		return nil, fmt.Errorf("weapon composition contains dependency cycle")
	}
	return order, nil
}

// graph returns each component's dependency count and its dependents.
func (c WeaponComposition) graph() (map[string]int, map[string][]string) {
	indegree := make(map[string]int, len(c.Components))
	dependents := make(map[string][]string, len(c.Components))
	for _, component := range c.Components {
		indegree[component.ID] = len(component.DependsOn)
		for _, dependency := range component.DependsOn {
			dependents[dependency] = append(dependents[dependency], component.ID)
		}
	}
	return indegree, dependents
}

// kahnOrder consumes indegree; components that become ready together are
// released in lexical order.
func kahnOrder(indegree map[string]int, dependents map[string][]string) []string {
	ready := make([]string, 0, len(indegree))
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	order := make([]string, 0, len(indegree))
	for len(ready) > 0 {
		id := ready[0]
		order = append(order, id)
		ready = releaseDependents(ready[1:], dependents[id], indegree)
	}
	return order
}

// releaseDependents decrements each child's indegree and appends the ones
// that became ready, keeping ready sorted.
func releaseDependents(ready, children []string, indegree map[string]int) []string {
	children = append([]string(nil), children...)
	sort.Strings(children)
	for _, child := range children {
		indegree[child]--
		if indegree[child] == 0 {
			ready = append(ready, child)
			sort.Strings(ready)
		}
	}
	return ready
}
