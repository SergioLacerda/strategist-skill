package install

import (
	"fmt"
	"os"
)

// ReadUpstreamContentDigest reads catalogPath from a plain filesystem (the
// maintainer/CI path — like PrepareEmbedded, not the compiled-in embed.FS)
// and returns the pinned upstream_content_digest of the provider with the
// given id. found is false when the provider exists but declares no pin
// (native roles and packages whose provenance was never researched — see
// validateVerifiedUpstreamProvenance). An unknown provider id is an error:
// the caller asked about a Weapon this catalog does not know, which is a
// different condition than "known but unpinned".
//
// This is the read-only counterpart task 3.4 needs to compare a Ranger's
// self-reported weapon_invocation.resolved_digest against the catalog's
// pin, without invoking anything.
func ReadUpstreamContentDigest(catalogPath, providerID string) (digest string, found bool, err error) {
	data, err := os.ReadFile(catalogPath) //nolint:gosec // G304: catalogPath is a caller-supplied path to a maintainer/CI-owned file, same trust model as PrepareEmbedded
	if err != nil {
		return "", false, fmt.Errorf("read upstream content digest: %w", err)
	}
	catalog, err := parseCatalogBytes(data)
	if err != nil {
		return "", false, fmt.Errorf("read upstream content digest: %w", err)
	}
	for _, provider := range catalog.Providers {
		if provider.ID == providerID {
			return provider.UpstreamContentDigest, provider.UpstreamContentDigest != "", nil
		}
	}
	return "", false, fmt.Errorf("read upstream content digest: no provider %q in %s", providerID, catalogPath)
}
