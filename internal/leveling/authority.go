package leveling

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// AuthorityClass identifies the result of checking the installed LEVELING
// runtime against the defaults carried by the executable.
type AuthorityClass string

// Authority classifications returned by the shared runtime decision.
const (
	AuthorityCurrent          AuthorityClass = "current"
	AuthorityLegacyCompatible AuthorityClass = "legacy_compatible"
	AuthorityMissing          AuthorityClass = "missing"
	AuthorityStale            AuthorityClass = "stale"
	AuthorityUnreadable       AuthorityClass = "unreadable"
)

// AuthorityDecision is the shared, surface-neutral result consumed by CLI and
// wizard activation. ManifestLoaded distinguishes an explicit current install
// from a legacy workspace that has no policy identity yet.
type AuthorityDecision struct {
	Class          AuthorityClass
	ManifestLoaded bool
}

// LoadAuthorized merges the customer override only after the embedded policy
// and install authority have been checked. Both CLI and wizard use this entry
// point so I/O, parsing, and compatibility semantics cannot drift.
func LoadAuthorized(root string, defaults, override []byte, source string) (EffectivePolicy, AuthorityDecision, error) {
	defaultPolicy, err := Parse(defaults)
	if err != nil {
		return EffectivePolicy{}, AuthorityDecision{Class: AuthorityStale}, fmt.Errorf("leveling_policy_stale: parse authoritative defaults: %w", err)
	}
	decision, err := evaluateAuthority(root, defaultPolicy, defaults, override)
	if err != nil {
		decision.Class = classifyAuthorityError(err)
		return EffectivePolicy{}, decision, err
	}
	effective, err := LoadEffective(defaults, override, source)
	if err != nil {
		return EffectivePolicy{}, AuthorityDecision{Class: AuthorityStale, ManifestLoaded: decision.ManifestLoaded}, fmt.Errorf("leveling: load effective policy: %w", err)
	}
	return effective, decision, nil
}

func classifyAuthorityError(err error) AuthorityClass {
	message := err.Error()
	if strings.Contains(message, "leveling_policy_missing") {
		return AuthorityMissing
	}
	if strings.Contains(message, "read install authority") {
		return AuthorityUnreadable
	}
	return AuthorityStale
}

func evaluateAuthority(root string, defaultsPolicy Policy, defaults, override []byte) (AuthorityDecision, error) {
	manifest, loaded, err := readAuthorityManifest(root)
	if err != nil {
		return AuthorityDecision{}, err
	}
	if loaded && strings.TrimSpace(manifest.LevelingPolicyDigest) != "" {
		if err := verifyManifestAuthority(defaultsPolicy, manifest); err != nil {
			return AuthorityDecision{}, err
		}
		return AuthorityDecision{Class: AuthorityCurrent, ManifestLoaded: true}, nil
	}
	if bytesMatch(defaults, override) {
		return AuthorityDecision{Class: AuthorityCurrent, ManifestLoaded: loaded}, nil
	}
	if err := requireLegacyCompatibility(root); err != nil {
		return AuthorityDecision{}, err
	}
	return AuthorityDecision{Class: AuthorityLegacyCompatible, ManifestLoaded: loaded}, nil
}

func readAuthorityManifest(root string) (domain.InstallManifest, bool, error) {
	path := filepath.Join(root, domain.InstallManifestRelPath)
	raw, err := os.ReadFile(path) //nolint:gosec // root is the discovered .strategist directory
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.InstallManifest{}, false, nil
		}
		return domain.InstallManifest{}, false, fmt.Errorf("leveling_policy_stale: read install authority: %w", err)
	}
	var manifest domain.InstallManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return domain.InstallManifest{}, false, fmt.Errorf("leveling_policy_stale: parse install authority: %w", err)
	}
	return manifest, true, nil
}

func verifyManifestAuthority(defaults Policy, manifest domain.InstallManifest) error {
	if manifest.LevelingPolicyVersion != defaults.Version || strings.TrimSpace(manifest.LevelingPolicyDigest) != defaults.Digest() {
		return fmt.Errorf("leveling_policy_stale: embedded policy authority differs from install manifest (expected version=%d digest=%s, observed version=%d digest=%s)", manifest.LevelingPolicyVersion, manifest.LevelingPolicyDigest, defaults.Version, defaults.Digest())
	}
	return nil
}

func bytesMatch(defaults, override []byte) bool {
	return domain.SHA256Hex(defaults) == domain.SHA256Hex(override)
}

func requireLegacyCompatibility(root string) error {
	path := filepath.Join(root, ".leveling-compat.yaml")
	raw, err := os.ReadFile(path) //nolint:gosec // root is the discovered .strategist directory
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("leveling_policy_missing: leveling.yaml is absent or an untracked override; reinstall or add versioned compatibility marker %s", path)
		}
		return fmt.Errorf("leveling_policy_stale: read compatibility marker: %w", err)
	}
	var marker struct {
		Version int    `yaml:"version"`
		Mode    string `yaml:"mode"`
	}
	if err := yaml.Unmarshal(raw, &marker); err != nil || marker.Version != 1 || strings.ToLower(strings.TrimSpace(marker.Mode)) != "legacy" {
		return fmt.Errorf("leveling_policy_stale: compatibility marker must declare version: 1 and mode: legacy")
	}
	return nil
}

// ValidateLegacyCompatibility validates the explicit marker used by callers
// that have no embedded extractor (for example legacy white-box fixtures).
// Production consumers should prefer LoadAuthorized so policy and authority
// are evaluated together.
func ValidateLegacyCompatibility(root string) error {
	return requireLegacyCompatibility(root)
}
