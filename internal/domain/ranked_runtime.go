package domain

import (
	"fmt"
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
}

const (
	// RankedRuntimeNone marks a provider that needs no private runtime.
	RankedRuntimeNone = "none"
	// RankedRuntimeOpenSpecRoot marks a provider backed by an initialized OpenSpec root.
	RankedRuntimeOpenSpecRoot = "openspec_root"
)

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
	if runtime.Root != "" || runtime.Bootstrap != "" || runtime.Healthcheck != "" {
		return fmt.Errorf("runtime kind %q cannot declare root, bootstrap, or healthcheck", runtime.Kind)
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
	return nil
}

func isSafeRuntimeRoot(root string) bool {
	if root == "" || root == "." || root == ".." {
		return false
	}
	if filepath.IsAbs(root) || filepath.Clean(root) != root {
		return false
	}
	root = filepath.ToSlash(root)
	return !strings.HasPrefix(root, "../") && strings.HasPrefix(root, ".strategist/")
}
