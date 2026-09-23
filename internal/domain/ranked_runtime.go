package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// RankedRuntimeContract describes the private runtime a build-certified
// Ranked provider needs before it can be invoked.
type RankedRuntimeContract struct {
	Kind        string `yaml:"kind"`
	Root        string `yaml:"root,omitempty"`
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
	case RankedRuntimeOpenSpecRoot:
		return validateOpenSpecRuntime(normalized)
	default:
		return fmt.Errorf("unsupported ranked runtime kind %q", r.Kind)
	}
}

func validateNoRuntime(runtime RankedRuntimeContract) error {
	if runtime.Root != "" || runtime.Bootstrap != "" || runtime.Healthcheck != "" || runtime.Version != "" || runtime.NodeVersion != "" {
		return fmt.Errorf("runtime kind %q cannot declare root, bootstrap, healthcheck, or pinned versions", runtime.Kind)
	}
	return nil
}

func validateOpenSpecRuntime(runtime RankedRuntimeContract) error {
	if !isSafeRuntimeRoot(runtime.Root) {
		return fmt.Errorf("openspec runtime root must be a clean relative path under .strategist, got %q", runtime.Root)
	}
	if runtime.Bootstrap == "" || runtime.Healthcheck == "" {
		return fmt.Errorf("openspec runtime requires bootstrap and healthcheck")
	}
	for name, value := range map[string]string{"version": runtime.Version, "node_version": runtime.NodeVersion} {
		if value != "" && !pinnedVersion.MatchString(value) {
			return fmt.Errorf("openspec runtime %s must be an exact MAJOR.MINOR.PATCH version, got %q", name, value)
		}
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

// ValidateOpenSpecHealthcheck verifies the semantic root reported by OpenSpec
// against the physical provider runtime directory. OpenSpec keeps its config
// under .strategist/openspec while context resolves the containing
// .strategist directory as the semantic root.
func ValidateOpenSpecHealthcheck(output []byte, runtimeRoot string) error {
	var contextResult struct {
		Root struct {
			Path string `json:"path"`
		} `json:"root"`
	}
	payload := output
	if start := bytes.IndexByte(output, '{'); start >= 0 {
		if end := bytes.LastIndexByte(output, '}'); end > start {
			payload = output[start : end+1]
		}
	}
	if err := json.Unmarshal(payload, &contextResult); err != nil {
		return fmt.Errorf("parse OpenSpec context: %w", err)
	}
	if contextResult.Root.Path == "" {
		return fmt.Errorf("OpenSpec context did not report root.path")
	}
	expected := canonicalRuntimePath(filepath.Dir(filepath.Clean(runtimeRoot)))
	observed := canonicalRuntimePath(contextResult.Root.Path)
	if observed != expected && !sameDirectory(expected, observed) {
		return fmt.Errorf("OpenSpec semantic root mismatch: expected %s, got %s (resolved paths differ; verify --root points at the prepared .strategist directory)", expected, observed)
	}
	return nil
}

// canonicalRuntimePath makes a path comparable regardless of spelling:
// relative, dotted, trailing-slash, or symlinked forms resolve to the same
// absolute physical path. A path that cannot be resolved (for example one
// that does not exist) falls back to its cleaned absolute form.
func canonicalRuntimePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// sameDirectory covers filesystems where two canonical spellings still name
// one directory (case-insensitive volumes, bind mounts).
func sameDirectory(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return false
	}
	return os.SameFile(infoA, infoB)
}
