package jeweler

import (
	"context"
	"encoding/json"
	"fmt"
)

type externalSearchResponse struct {
	Envelope struct {
		Provider      string   `json:"Provider"`
		Version       string   `json:"Version"`
		SchemaVersion int      `json:"SchemaVersion"`
		Freshness     string   `json:"Freshness"`
		FallbackState string   `json:"FallbackState"`
		Limitations   []string `json:"Limitations"`
	} `json:"Envelope"`
	Items *[]Item `json:"Items"`
}

// Query invokes the external provider's bounded search operation.
func (p CLIProvider) Query(ctx context.Context, req QueryRequest) (Projection, error) {
	run, err := p.runner()
	if err != nil {
		return Projection{}, err
	}
	data, err := run(ctx, p.Executable, searchArgs(req)...)
	if err != nil {
		return Projection{}, fmt.Errorf("external search: %w", err)
	}
	return decodeSearchResponse(data, p.Identity)
}

func searchArgs(req QueryRequest) []string {
	args := []string{"search", "--root", req.Root, "--json", "--token-budget", fmt.Sprint(req.Limit)}
	if req.Intent != "" {
		args = append(args, "--intent", req.Intent)
	}
	args = appendRepeatedArg(args, "--concept", req.Concepts)
	return appendRepeatedArg(args, "--symbol", req.Symbols)
}

func appendRepeatedArg(args []string, name string, values []string) []string {
	for _, value := range values {
		args = append(args, name, value)
	}
	return args
}

func decodeSearchResponse(data []byte, identity ProviderIdentity) (Projection, error) {
	var result externalSearchResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return Projection{}, fmt.Errorf("external search response: %w", err)
	}
	if err := validateSearchResponse(result, identity); err != nil {
		return Projection{}, err
	}
	freshness, err := externalFreshness(result.Envelope.Freshness)
	if err != nil {
		return Projection{}, err
	}
	return Projection{ContractVersion: ContractVersion, Identity: identity, Freshness: freshness, Items: *result.Items, Limitations: result.Envelope.Limitations}, nil
}

func validateSearchResponse(result externalSearchResponse, identity ProviderIdentity) error {
	if result.Items == nil {
		return fmt.Errorf("external search response: items are missing")
	}
	if result.Envelope.Provider != identity.Package || result.Envelope.Version != identity.Version {
		return fmt.Errorf("external search response: envelope provider identity mismatch")
	}
	if result.Envelope.SchemaVersion != KnowledgeAPIVersion {
		return fmt.Errorf("external search response: schema version %d is unsupported", result.Envelope.SchemaVersion)
	}
	if result.Envelope.FallbackState != "none" {
		return fmt.Errorf("external search response: fallback state %q is not allowed", result.Envelope.FallbackState)
	}
	return nil
}

func externalFreshness(value string) (Freshness, error) {
	switch value {
	case "fresh":
		return FreshnessFresh, nil
	case "stale", "partially_stale":
		return FreshnessStale, nil
	case "unknown":
		return FreshnessUnknown, nil
	default:
		return FreshnessUnknown, fmt.Errorf("external search response: freshness %q is unsupported", value)
	}
}
