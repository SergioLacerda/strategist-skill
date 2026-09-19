package mission

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// ContextIdentityInput contains every digest that can affect materialization.
type ContextIdentityInput struct {
	SourceDigest   string
	RuntimeDigest  string
	ProviderDigest string
	PolicyDigest   string
	ContentDigest  string
}

// ContextIdentity is the provenance-preserving address of reusable context.
type ContextIdentity struct {
	SourceDigest   string `json:"source_digest"`
	RuntimeDigest  string `json:"runtime_digest"`
	ProviderDigest string `json:"provider_digest"`
	PolicyDigest   string `json:"policy_digest"`
	ContentDigest  string `json:"content_digest"`
	Key            string `json:"key"`
}

// NewContextIdentity validates and hashes the canonical digest envelope.
func NewContextIdentity(input ContextIdentityInput) (ContextIdentity, error) {
	identity := ContextIdentity{
		SourceDigest: input.SourceDigest, RuntimeDigest: input.RuntimeDigest,
		ProviderDigest: input.ProviderDigest, PolicyDigest: input.PolicyDigest,
		ContentDigest: input.ContentDigest,
	}
	if err := identity.validate(); err != nil {
		return ContextIdentity{}, err
	}
	data, err := json.Marshal(identity)
	if err != nil {
		return ContextIdentity{}, fmt.Errorf("context identity: marshal: %w", err)
	}
	sum := sha256.Sum256(data)
	identity.Key = "sha256:" + hex.EncodeToString(sum[:])
	return identity, nil
}

func (i ContextIdentity) validate() error {
	fields := []struct {
		name  string
		value string
	}{
		{name: "source_digest", value: i.SourceDigest},
		{name: "runtime_digest", value: i.RuntimeDigest},
		{name: "provider_digest", value: i.ProviderDigest},
		{name: "policy_digest", value: i.PolicyDigest},
		{name: "content_digest", value: i.ContentDigest},
	}
	for _, field := range fields {
		if field.value == "" {
			return fmt.Errorf("context identity: %s is required", field.name)
		}
	}
	return nil
}
