package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrationEventName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		resolutionError string
		want            string
	}{
		{"resolved", "", "resolved"},
		{"unresolved active slot", "unresolved_active_slot: discovery has empty provider", "unresolved_active_slot"},
		{"id shadowing", "id_shadowing: brainstorming shadows another candidate", "id_shadowing"},
		{"role binding missing", "role_binding_missing: no candidate for role", "role_binding_missing"},
		{"role binding ambiguous", "role_binding_ambiguous: more than one candidate", "role_binding_ambiguous"},
		{"unrecognized reason", "something else entirely", "role_binding_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := RoleProviderPreviewEntry{ResolutionError: tt.resolutionError}
			assert.Equal(t, tt.want, migrationEventName(entry))
		})
	}
}
