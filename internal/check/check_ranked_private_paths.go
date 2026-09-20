package check

import (
	"path"
	"path/filepath"
	"strings"
)

func privateRuntimePaths(root string, private rankedRuntimeStatePrivate) (node, script string, ok bool) {
	const prefix = "weapon-runtime/"
	for _, rel := range []string{private.Node, private.Script} {
		if !strings.HasPrefix(rel, prefix) || path.Clean(rel) != rel || strings.Contains(rel, `\`) || strings.Contains(rel, ":") {
			return "", "", false
		}
	}
	return filepath.Join(root, filepath.FromSlash(private.Node)), filepath.Join(root, filepath.FromSlash(private.Script)), true
}
