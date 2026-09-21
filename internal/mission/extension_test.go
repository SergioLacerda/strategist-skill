package mission

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateExtensionRegistration(t *testing.T) {
	valid := ExtensionRegistration{ID: ContextMaterializationExtension, Version: "v1", Slot: "discovery", Authority: "strategist", Rollback: "disable"}
	require.NoError(t, ValidateExtensionRegistration(valid))
	valid.Slot = "execution"
	require.Error(t, ValidateExtensionRegistration(valid))
	valid.Slot, valid.Version = "discovery", ""
	require.Error(t, ValidateExtensionRegistration(valid))
}

func TestExecuteExtensionRecoversLastKnownGood(t *testing.T) {
	registration := ExtensionRegistration{ID: ContextMaterializationExtension, Version: "v1", Slot: "discovery", Authority: "strategist", Rollback: "disable"}
	result, err := ExecuteExtension(registration, "known-good", func() (string, error) { return "", errors.New("failed") })
	require.ErrorContains(t, err, "failed")
	require.Equal(t, ExtensionResult{Value: "known-good", Recovered: true}, result)
}

func TestValidateExtensionRegistrationRejectsEachInvalidField(t *testing.T) {
	base := ExtensionRegistration{ID: ContextMaterializationExtension, Version: "v1", Slot: "discovery", Authority: "strategist", Rollback: "disable"}
	tests := []struct {
		name   string
		mutate func(*ExtensionRegistration)
		want   string
	}{
		{"unknown id", func(r *ExtensionRegistration) { r.ID = "arbitrary_phase" }, "unknown or out-of-scope"},
		{"other slot", func(r *ExtensionRegistration) { r.Slot = "refinement" }, "unknown or out-of-scope"},
		{"unversioned", func(r *ExtensionRegistration) { r.Version = "" }, "version, authority, and rollback"},
		{"unsupported version", func(r *ExtensionRegistration) { r.Version = "v2" }, "version, authority, and rollback"},
		{"missing authority", func(r *ExtensionRegistration) { r.Authority = "" }, "version, authority, and rollback"},
		{"missing rollback", func(r *ExtensionRegistration) { r.Rollback = "" }, "version, authority, and rollback"},
		{"declares write scope", func(r *ExtensionRegistration) { r.WriteScope = "docs/**" }, "cannot declare write scope"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			tc.mutate(&r)
			require.ErrorContains(t, ValidateExtensionRegistration(r), tc.want)
		})
	}
}

func TestExecuteExtensionDoesNotRunInvalidRegistrationAndHandlesMissingExecutor(t *testing.T) {
	ran := false
	_, err := ExecuteExtension(ExtensionRegistration{ID: "arbitrary_phase"}, "lkg", func() (string, error) { ran = true; return "x", nil })
	require.Error(t, err)
	require.False(t, ran, "an unregistered extension must never execute")

	valid := ExtensionRegistration{ID: ContextMaterializationExtension, Version: "v1", Slot: "discovery", Authority: "strategist", Rollback: "disable"}
	result, err := ExecuteExtension(valid, "lkg", nil)
	require.ErrorContains(t, err, "executor is required")
	require.Equal(t, ExtensionResult{Value: "lkg", Recovered: true}, result)

	result, err = ExecuteExtension(valid, "lkg", func() (string, error) { return "fresh", nil })
	require.NoError(t, err)
	require.Equal(t, ExtensionResult{Value: "fresh"}, result)
}
