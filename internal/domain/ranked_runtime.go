package domain

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// RankedRuntimeContract describes the private runtime a build-certified
// Ranked provider needs before it can be invoked.
type RankedRuntimeContract struct {
	Kind        string `yaml:"kind"`
	HostAPI     string `yaml:"host_api,omitempty"`
	Root        string `yaml:"root,omitempty"`
	Entrypoint  string `yaml:"entrypoint,omitempty"`
	Bootstrap   string `yaml:"bootstrap,omitempty"`
	Healthcheck string `yaml:"healthcheck,omitempty"`
	// Version and NodeVersion pin the runtime identity (the provider CLI and
	// the Node it runs on). Optional, but when present they are part of the
	// certification digest: changing either re-certifies the provider.
	Version     string `yaml:"version,omitempty"`
	NodeVersion string `yaml:"node_version,omitempty"`
}

const (
	// RankedRuntimeNone marks a provider that needs no private runtime.
	RankedRuntimeNone = "none"
	// RankedRuntimeOpenSpecRoot marks a provider backed by an initialized OpenSpec root.
	RankedRuntimeOpenSpecRoot = "openspec_root"
	// RankedRuntimeHost marks a host-provided Weapon invocation boundary.
	RankedRuntimeHost = "host"
	// RankedRuntimeEmbedded marks a Strategist-owned in-process Weapon runtime.
	// Unlike host, it does not require a host API or an external loader.
	RankedRuntimeEmbedded = "embedded"
	// RankedRuntimeExecutable marks an executable invocation boundary.
	RankedRuntimeExecutable = "executable"
	// MinimumOpenSpecNodeVersion is the upstream engine requirement for the
	// OpenSpec bundle embedded in ordinary Strategist builds.
	MinimumOpenSpecNodeVersion = "20.19.0"
)

// ReasonRankedRuntimeExecutableMissing is the cataloged reason code emitted
// when a Ranked provider's executable cannot be found (see
// machine/errors.yaml).
const ReasonRankedRuntimeExecutableMissing = "ranked_runtime_executable_missing"

// RankedRuntimeExecutableMissingMessage explains a missing Node executable in
// operator terms. Every binary embeds OpenSpec; payload builds embed Node too,
// while ordinary go-install builds use the supported host Node runtime.
func RankedRuntimeExecutableMissingMessage(provider, executable string) string {
	return fmt.Sprintf("Ranked provider %q needs the %q executable to run its embedded OpenSpec bundle, but it was not found. "+
		"Install Node.js >=20.19.0 and rerun `strategist install --wizard` (keep the Ranked option); OpenSpec and npm do not need to be installed separately. "+
		"See docs/runbooks/standalone-runtime-hermeticity.md",
		provider, executable)
}

// NormalizeRankedRuntime makes an omitted runtime declaration explicit.
func NormalizeRankedRuntime(runtime RankedRuntimeContract) RankedRuntimeContract {
	if strings.TrimSpace(runtime.Kind) == "" {
		runtime.Kind = RankedRuntimeNone
	}
	return runtime
}

// Validate checks that the runtime kind and its paths/commands are safe and complete.
func (r RankedRuntimeContract) Validate() error {
	normalized := NormalizeRankedRuntime(r)
	switch normalized.Kind {
	case RankedRuntimeNone:
		return validateNoRuntime(normalized)
	case RankedRuntimeHost:
		return validateHostRuntime(normalized)
	case RankedRuntimeEmbedded:
		return validateEmbeddedRuntime(normalized)
	case RankedRuntimeExecutable:
		return validateExecutableRuntime(normalized)
	case RankedRuntimeOpenSpecRoot:
		return validateOpenSpecRuntime(normalized)
	default:
		return unsupportedRuntimeKindError(r.Kind)
	}
}

func validateNoRuntime(runtime RankedRuntimeContract) error {
	if runtime.HostAPI != "" || runtime.Root != "" || runtime.Entrypoint != "" || runtime.Bootstrap != "" || runtime.Healthcheck != "" || runtime.Version != "" || runtime.NodeVersion != "" {
		return fmt.Errorf("runtime kind %q cannot declare invocation or runtime fields", runtime.Kind)
	}
	return nil
}

func validateHostRuntime(runtime RankedRuntimeContract) error {
	if strings.TrimSpace(runtime.HostAPI) == "" {
		return fmt.Errorf("host runtime requires host_api")
	}
	if runtime.Root != "" || runtime.Entrypoint != "" || runtime.Bootstrap != "" || runtime.Healthcheck != "" || runtime.Version != "" || runtime.NodeVersion != "" {
		return fmt.Errorf("host runtime cannot declare executable or OpenSpec fields")
	}
	return nil
}

func validateEmbeddedRuntime(runtime RankedRuntimeContract) error {
	if runtime.HostAPI != "" || runtime.Root != "" || runtime.Entrypoint != "" || runtime.Bootstrap != "" || runtime.Healthcheck != "" || runtime.Version != "" || runtime.NodeVersion != "" {
		return fmt.Errorf("embedded runtime cannot declare host, executable, or external runtime fields")
	}
	return nil
}

func validateExecutableRuntime(runtime RankedRuntimeContract) error {
	if strings.TrimSpace(runtime.Entrypoint) == "" {
		return fmt.Errorf("executable runtime requires entrypoint")
	}
	if runtime.HostAPI != "" || runtime.Root != "" || runtime.Bootstrap != "" || runtime.Healthcheck != "" || runtime.Version != "" || runtime.NodeVersion != "" {
		return fmt.Errorf("executable runtime cannot declare host or OpenSpec fields")
	}
	return nil
}

func validateOpenSpecRuntime(runtime RankedRuntimeContract) error {
	if runtime.HostAPI != "" || runtime.Entrypoint != "" {
		return fmt.Errorf("openspec runtime cannot declare host or executable fields")
	}
	if !isSafeRuntimeRoot(runtime.Root) {
		return fmt.Errorf("openspec runtime root must be a clean relative path under .strategist, got %q", runtime.Root)
	}
	if runtime.Bootstrap == "" || runtime.Healthcheck == "" {
		return fmt.Errorf("openspec runtime requires bootstrap and healthcheck")
	}
	return validatePinnedVersions(runtime)
}

func validatePinnedVersions(runtime RankedRuntimeContract) error {
	for name, value := range map[string]string{"version": runtime.Version, "node_version": runtime.NodeVersion} {
		if value != "" && !pinnedVersion.MatchString(value) {
			return fmt.Errorf("openspec runtime %s must be an exact MAJOR.MINOR.PATCH version, got %q", name, value)
		}
	}
	return nil
}

// ValidateActive requires an explicit runtime for an active Weapon. Static
// metadata alone is never sufficient to authorize invocation.
func (r RankedRuntimeContract) ValidateActive() error {
	if err := r.Validate(); err != nil {
		return err
	}
	if NormalizeRankedRuntime(r).Kind == RankedRuntimeNone {
		return fmt.Errorf("active Weapon requires an invocable runtime")
	}
	return nil
}

// isSafeRuntimeRoot accepts only a canonical slash-separated path under
// .strategist. The separator is normalized before the cleanliness comparison:
// on Windows filepath.Clean rewrites "/" to "\\", so comparing the raw string
// against its cleaned form would reject the catalog's own declaration.
func isSafeRuntimeRoot(root string) bool {
	if root == "" || filepath.IsAbs(root) {
		return false
	}
	slash := filepath.ToSlash(root)
	if !hasSafeRuntimePrefix(slash) {
		return false
	}
	return safeRuntimeSegments(slash)
}

func hasSafeRuntimePrefix(slash string) bool {
	return !path.IsAbs(slash) && path.Clean(slash) == slash && strings.HasPrefix(slash, ".strategist/")
}

func safeRuntimeSegments(slash string) bool {
	for _, segment := range strings.Split(slash, "/") {
		if segment == ".." || strings.Contains(segment, ":") {
			return false
		}
	}
	return true
}
