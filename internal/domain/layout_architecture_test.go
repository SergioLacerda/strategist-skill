//go:build integration

package domain_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const layoutModulePath = "github.com/SergioLacerda/strategist-skill"

var layoutNamespaces = []string{"feats", "tools", "roles", "pipelines"}

// TestNamespaceParentsAreNotImplementationPackages keeps taxonomy namespaces
// as directories only. Every implementation must live in an independently
// testable child package, never directly under internal/feats, tools, roles,
// or pipelines.
func TestNamespaceParentsAreNotImplementationPackages(t *testing.T) {
	t.Parallel()
	root := layoutRepositoryRoot(t)
	for _, namespace := range layoutNamespaces {
		assertNamespaceParentIsDirectory(t, root, namespace)
	}
}

func assertNamespaceParentIsDirectory(t *testing.T, root, namespace string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "internal", namespace))
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("read %s namespace: %v", namespace, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			t.Errorf("internal/%s must remain a namespace; implementation file %s is at the parent level", namespace, entry.Name())
		}
	}
}

// TestNamespaceChildrenDoNotImportSiblingImplementations makes lateral
// coupling explicit for the namespace pattern. Shared contracts belong in a
// domain or port package rather than in a sibling Feat/Tool/Role package.
func TestNamespaceChildrenDoNotImportSiblingImplementations(t *testing.T) {
	t.Parallel()
	root := layoutRepositoryRoot(t)
	for _, namespace := range layoutNamespaces {
		assertNamespaceChildrenDoNotImportSiblings(t, root, namespace)
	}
}

// TestApplicationChildrenRespectBoundary keeps application use-case packages
// independent from UI, telemetry, providers, and concrete plugin adapters.
// The parent package and every extracted child are checked so a new use-case
// package cannot quietly bypass the existing composition boundary.
func TestApplicationChildrenRespectBoundary(t *testing.T) {
	t.Parallel()
	root := layoutRepositoryRoot(t)
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}", "./internal/application/...")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list application packages failed: %v\n%s", err, out)
	}
	for _, pkg := range strings.Fields(string(out)) {
		assertNoForbiddenDeps(t, pkg, []string{
			layoutModulePath + "/cmd/strategist",
			layoutModulePath + "/internal/telemetry",
			layoutModulePath + "/internal/provider",
			layoutModulePath + "/internal/plugins",
			"github.com/spf13/cobra",
		})
	}
}

// TestDomainChildrenRespectBoundary keeps extracted domain aggregates pure:
// they may depend on the parent domain contract package, but never on runtime,
// adapter, UI, or application layers.
func TestDomainChildrenRespectBoundary(t *testing.T) {
	t.Parallel()
	root := layoutRepositoryRoot(t)
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}", "./internal/domain/...")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list domain packages failed: %v\n%s", err, out)
	}
	for _, pkg := range strings.Fields(string(out)) {
		assertNoForbiddenDeps(t, pkg, []string{
			layoutModulePath + "/cmd/strategist",
			layoutModulePath + "/internal/application",
			layoutModulePath + "/internal/telemetry",
			layoutModulePath + "/internal/provider",
			layoutModulePath + "/internal/plugins",
			"github.com/spf13/cobra",
		})
	}
}

type layoutPackageImports struct {
	path    string
	imports []string
}

func assertNamespaceChildrenDoNotImportSiblings(t *testing.T, root, namespace string) {
	t.Helper()
	packages, ok := listNamespacePackages(t, root, namespace)
	if !ok {
		return
	}
	prefix := layoutModulePath + "/internal/" + namespace + "/"
	parent := layoutModulePath + "/internal/" + namespace
	for _, pkg := range packages {
		if pkg.path == parent {
			continue
		}
		assertNoSiblingImports(t, prefix, pkg)
	}
}

func listNamespacePackages(t *testing.T, root, namespace string) ([]layoutPackageImports, bool) {
	t.Helper()
	pattern := "./internal/" + namespace + "/..."
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}|{{join .Imports \" \"}}", pattern)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, statErr := os.Stat(filepath.Join(root, "internal", namespace)); os.IsNotExist(statErr) {
			return nil, false
		}
		t.Fatalf("go list %s: %v\n%s", pattern, err, out)
	}
	return parseNamespacePackages(t, namespace, string(out)), true
}

func parseNamespacePackages(t *testing.T, namespace, output string) []layoutPackageImports {
	t.Helper()
	var packages []layoutPackageImports
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			t.Fatalf("unexpected go list output for %s: %q", namespace, line)
		}
		packages = append(packages, layoutPackageImports{path: parts[0], imports: strings.Fields(parts[1])})
	}
	return packages
}

func assertNoSiblingImports(t *testing.T, prefix string, pkg layoutPackageImports) {
	t.Helper()
	for _, imported := range pkg.imports {
		if strings.HasPrefix(imported, prefix) && imported != pkg.path {
			t.Errorf("%s must not import sibling implementation package %s", pkg.path, imported)
		}
	}
}

func layoutRepositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}
