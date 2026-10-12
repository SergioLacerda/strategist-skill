// Package treasuretmp exposes the root Treasure Chest implementation through
// the temporary JEWELER host protocol. No Strategist mission imports the root
// package; this adapter is the sole implementation boundary for Sprint 1.
package treasuretmp

import (
	"fmt"
	"strings"

	treasure "github.com/SergioLacerda/strategist-skill/treasure-chest"
)

const (
	// ProviderName identifies the temporary Treasure Chest Weapon.
	ProviderName = "treasure-chest-tmp"
	// ProviderVersion identifies the temporary Weapon release.
	ProviderVersion = "0.1.0"
	// SchemaVersion identifies the JSON protocol schema.
	SchemaVersion = 1
)

// Envelope identifies the provider and the freshness of a search response.
type Envelope struct {
	Provider      string   `json:"Provider"`
	Version       string   `json:"Version"`
	SchemaVersion int      `json:"SchemaVersion"`
	Freshness     string   `json:"Freshness"`
	FallbackState string   `json:"FallbackState"`
	Limitations   []string `json:"Limitations,omitempty"`
}

// Source identifies the source evidence for a returned item.
type Source struct {
	Path   string `json:"Path"`
	Digest string `json:"Digest,omitempty"`
}

// Item is one Jewel or Potion returned by the temporary provider.
type Item struct {
	ID            string  `json:"ID"`
	Kind          string  `json:"Kind"`
	Score         float64 `json:"Score"`
	Trust         float64 `json:"Trust"`
	Freshness     string  `json:"Freshness"`
	Applicability string  `json:"Applicability"`
	Reason        string  `json:"Reason"`
	Source        Source  `json:"Source"`
}

// SearchResponse is the bounded response for a provider search.
type SearchResponse struct {
	Envelope Envelope `json:"Envelope"`
	Items    []Item   `json:"Items"`
}

// PrepareResponse reports the indexed sources and items.
type PrepareResponse struct {
	Envelope     Envelope `json:"Envelope"`
	SourcesRead  int      `json:"SourcesRead"`
	ItemsIndexed int      `json:"ItemsIndexed"`
}

// StatusResponse reports temporary provider health.
type StatusResponse struct {
	Provider      string   `json:"Provider"`
	Version       string   `json:"Version"`
	SchemaVersion int      `json:"SchemaVersion"`
	Healthy       bool     `json:"Healthy"`
	Limitations   []string `json:"Limitations,omitempty"`
}

// Request carries the bounded provider search inputs.
type Request struct {
	Root        string
	Intent      string
	Concepts    []string
	Symbols     []string
	TokenBudget int
}

type index struct {
	governed map[string]treasure.GovernedChest
	jewels   map[string][]treasure.Jewel
	potions  map[string][]treasure.Potion
}

func envelope(limitations ...string) Envelope {
	return Envelope{
		Provider: ProviderName, Version: ProviderVersion, SchemaVersion: SchemaVersion,
		Freshness: "fresh", FallbackState: "none", Limitations: limitations,
	}
}

func load(root string) (index, error) {
	if strings.TrimSpace(root) == "" {
		return index{}, fmt.Errorf("treasure-chest-tmp: root is required")
	}
	governed, err := treasure.LoadGoverned(root)
	if err != nil {
		return index{}, fmt.Errorf("load governed chests: %w", err)
	}
	if err := validateRunbookSourceConsistency(root, governed); err != nil {
		return index{}, fmt.Errorf("treasure-chest-tmp: runbook source consistency: %w", err)
	}
	jewels, err := treasure.LoadJewels(root, governed)
	if err != nil {
		return index{}, fmt.Errorf("load jewels: %w", err)
	}
	potions, err := treasure.LoadPotions(root, governed)
	if err != nil {
		return index{}, fmt.Errorf("load potions: %w", err)
	}
	return index{governed: governed, jewels: jewels, potions: potions}, nil
}

// Status checks whether the temporary provider can load its governed sources.
func Status(root string) StatusResponse {
	status := StatusResponse{Provider: ProviderName, Version: ProviderVersion, SchemaVersion: SchemaVersion}
	if _, err := load(root); err != nil {
		status.Limitations = []string{"unhealthy: " + err.Error()}
		return status
	}
	status.Healthy = true
	return status
}

// Prepare loads the governed sources and reports the resulting index size.
func Prepare(root string) (PrepareResponse, error) {
	loaded, err := load(root)
	if err != nil {
		return PrepareResponse{}, err
	}
	items := 0
	for _, list := range loaded.jewels {
		items += len(list)
	}
	for _, list := range loaded.potions {
		items += len(list)
	}
	return PrepareResponse{Envelope: envelope(), SourcesRead: len(loaded.governed), ItemsIndexed: items}, nil
}
