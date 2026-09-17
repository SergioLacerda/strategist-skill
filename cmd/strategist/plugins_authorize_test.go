package main

import (
	"fmt"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/authorization"
	"github.com/stretchr/testify/assert"
)

func TestPluginsAuthorizeCmd_IsRegistered(t *testing.T) {
	var found bool
	for _, command := range pluginsCmd.Commands() {
		if command.Use == "authorize" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected authorize to be registered under plugins")
}

func TestExitCodeForAuthorizationStates(t *testing.T) {
	assert.Equal(t, 2, exitCodeFor(fmt.Errorf("%w: denied", authorization.ErrDenied)))
}
