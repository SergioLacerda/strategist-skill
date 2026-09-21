package provider

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestMinimalFixtureProviderConformance(t *testing.T) {
	source, reasons := loadSource(fixturePath(t))
	require.Empty(t, reasons)
	require.NoError(t, source.Package.Validate())
	require.NoError(t, source.Adapter.Validate())
	require.Equal(t, source.Package.ID, source.Adapter.ID)
	require.Contains(t, source.Adapter.SupportedRoles, "archivist")
	require.Contains(t, source.Adapter.SupportedSlots, string(domain.SlotRefinement))
	require.Contains(t, source.Adapter.SupportedHandoffSchemas, "schemas/handoff-archivist-to-sniper.schema.yaml")
	require.Contains(t, source.Adapter.Capabilities, "analysis.read")
	for _, permission := range source.Adapter.RequestedPermissions {
		require.True(t, domain.IsKnownPluginPermission(permission))
	}
	report, err := Validate(fixturePath(t), string(domain.SlotRefinement))
	require.NoError(t, err)
	require.Equal(t, domain.ReadinessUnknown, report.Readiness.PermissionGrant.Status)
	require.Equal(t, domain.ReadinessUnknown, report.LiveInvocation.Status)
}
