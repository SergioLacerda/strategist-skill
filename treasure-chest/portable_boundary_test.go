package treasure_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestProviderPackagesHaveNoHostInternalImports keeps the provider core
// movable. The CLI adapter is intentionally excluded: it is the host-side
// integration surface, while the provider root and its domain package must be
// independently buildable in the external Weapon repository.
func TestProviderPackagesHaveNoHostInternalImports(t *testing.T) {
	t.Parallel()
	hostModule := modulePath(t)

	for _, pkg := range []string{".", "./domain"} {
		dependencies := packageDependencies(t, pkg)
		for _, imported := range dependencies {
			if strings.HasPrefix(imported, hostModule+"/internal/") {
				t.Errorf("provider package %s imports host-internal package %s", pkg, imported)
			}
		}
	}
}

func modulePath(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Path}}").CombinedOutput()
	if err != nil {
		t.Fatalf("go list module failed: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func packageDependencies(t *testing.T, pkg string) []string {
	t.Helper()
	out, err := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", pkg).CombinedOutput()
	if err != nil {
		t.Fatalf("go list %s failed: %v\n%s", pkg, err, out)
	}
	return strings.Fields(string(out))
}
