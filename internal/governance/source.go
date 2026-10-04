package governance

import (
	"fmt"
	"path/filepath"
	"sort"
)

// Source is the consumer-owned boundary for provisioned governance. A source
// is selected explicitly by its caller; it never discovers a provider path.
type Source interface {
	Name() string
	Snapshot(governanceDir string) (Snapshot, error)
}

// Snapshot is the normalized, read-only governance context consumed by sync
// and bridge code. Provider adapters own parsing; this package owns the
// invariants that make the result safe to consume.
type Snapshot struct {
	SourceID       string
	Fingerprint    string
	ActiveMandates []string
	Scope          string
	CorrelationID  string
	Validated      bool
}

func validateSnapshot(source Source, governanceDir string, snapshot Snapshot) (Snapshot, error) {
	name, err := validateSource(source, governanceDir, snapshot)
	if err != nil {
		return Snapshot{}, err
	}

	snapshot.SourceID = name
	snapshot.Scope = filepath.Clean(governanceDir)
	if snapshot.CorrelationID == "" {
		snapshot.CorrelationID = name + ":" + snapshot.Fingerprint
	}
	if snapshot.CorrelationID == "" {
		return Snapshot{}, fmt.Errorf("governance source %q returned an empty correlation", name)
	}
	snapshot.ActiveMandates, err = normalizeMandates(name, snapshot.ActiveMandates)
	if err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func validateSource(source Source, governanceDir string, snapshot Snapshot) (string, error) {
	if source == nil {
		return "", fmt.Errorf("governance source is required")
	}
	name := source.Name()
	if name == "" {
		return "", fmt.Errorf("governance source name is required")
	}
	if snapshot.SourceID != "" && snapshot.SourceID != name {
		return "", fmt.Errorf("governance source identity mismatch: %q != %q", snapshot.SourceID, name)
	}
	if snapshot.Fingerprint == "" {
		return "", fmt.Errorf("governance source %q returned an empty fingerprint", name)
	}
	if !snapshot.Validated {
		return "", fmt.Errorf("governance source %q returned an unvalidated snapshot", name)
	}
	if governanceDir == "" {
		return "", fmt.Errorf("governance directory is required")
	}
	return name, nil
}

func normalizeMandates(sourceName string, active []string) ([]string, error) {
	mandates := append([]string(nil), active...)
	sort.Strings(mandates)
	for i, mandate := range mandates {
		if mandate == "" {
			return nil, fmt.Errorf("governance source %q returned an empty mandate", sourceName)
		}
		if i > 0 && mandates[i-1] == mandate {
			return nil, fmt.Errorf("governance source %q returned duplicate mandate %q", sourceName, mandate)
		}
	}
	return mandates, nil
}
