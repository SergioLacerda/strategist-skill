package telemetry_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
)

func TestNewCompositeWeaponEventDoesNotContainRawPayload(t *testing.T) {
	event := telemetry.NewCompositeWeaponEvent("run-1", "ranger", "discovery", "suite", "brainstorming", "run-1/suite", "ready", "", "evidence-1", true, false)
	assert.Equal(t, telemetry.CompositeWeaponEventName, event.Name)
	assert.Equal(t, "suite", event.Attributes[telemetry.AttrWeapon])
	assert.Equal(t, "brainstorming", event.Attributes[telemetry.AttrWeaponComponent])
	assert.Equal(t, "evidence-1", event.Attributes[telemetry.AttrWeaponInvocationEvidence])
	assert.NotContains(t, event.Attributes, "artifact")
	assert.NotContains(t, event.Attributes, "payload")
}
