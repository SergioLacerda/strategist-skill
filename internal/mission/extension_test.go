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
