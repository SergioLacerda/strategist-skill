package main

import (
	"fmt"

	dojoadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
)

func dojoDependencies() dojoadapter.Dependencies {
	return dojoadapter.Dependencies{ResolveRoots: resolveDojoRoots}
}

// resolveDojoRoots delegates to internal/cliutil (shared with internal/treasurecli
// — see its own cli_bridge.go), re-adding the "dojo: " error prefix this
// command's callers have always received unwrapped.
func resolveDojoRoots(root string) (strategistRoot, basePath string, err error) {
	strategistRoot, basePath, err = cliutil.ResolveActiveBasePath(root)
	if err != nil {
		return "", "", fmt.Errorf("dojo: %w", err)
	}
	return strategistRoot, basePath, nil
}
