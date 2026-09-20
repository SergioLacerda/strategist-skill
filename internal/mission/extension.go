package mission

import "fmt"

// ContextMaterializationExtension is the only Phase D extension point.
const ContextMaterializationExtension = "context_materialization"

// ExtensionRegistration declares a bounded adapter in the fixed graph.
type ExtensionRegistration struct {
	ID         string
	Version    string
	Slot       string
	Authority  string
	WriteScope string
	Rollback   string
}

// ExtensionExecutor evaluates a bounded adapter without access to pipeline state.
type ExtensionExecutor func() (string, error)

// ExtensionResult reports isolated execution and the last-known-good result.
type ExtensionResult struct {
	Value     string
	Recovered bool
}

// ValidateExtensionRegistration rejects registrations outside discovery.
func ValidateExtensionRegistration(registration ExtensionRegistration) error {
	if registration.ID != ContextMaterializationExtension || registration.Slot != "discovery" {
		return fmt.Errorf("extension: unknown or out-of-scope registration")
	}
	if registration.Version != "v1" || registration.Authority == "" || registration.Rollback == "" {
		return fmt.Errorf("extension: version, authority, and rollback are required")
	}
	if registration.WriteScope != "" {
		return fmt.Errorf("extension: context materialization cannot declare write scope")
	}
	return nil
}

// ExecuteExtension preserves the last-known-good value when the adapter fails.
func ExecuteExtension(registration ExtensionRegistration, lastKnownGood string, execute ExtensionExecutor) (ExtensionResult, error) {
	if err := ValidateExtensionRegistration(registration); err != nil {
		return ExtensionResult{}, err
	}
	if execute == nil {
		return ExtensionResult{Value: lastKnownGood, Recovered: true}, fmt.Errorf("extension: executor is required")
	}
	value, err := execute()
	if err != nil {
		return ExtensionResult{Value: lastKnownGood, Recovered: true}, fmt.Errorf("extension: execute: %w", err)
	}
	return ExtensionResult{Value: value}, nil
}
