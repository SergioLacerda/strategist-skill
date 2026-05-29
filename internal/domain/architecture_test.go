package domain_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestDomainIsolation verifies that the domain package is a pure type/contract
// layer with no dependencies on other internal packages. This prevents
// architectural drift where business-logic packages accidentally import domain
// and create circular dependency risk.
func TestDomainIsolation(t *testing.T) {
	t.Parallel()

	out, err := exec.Command(
		"go", "list", "-deps",
		"github.com/SergioLacerda/strategist-skill/internal/domain",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\n%s", err, out)
	}

	forbidden := []string{
		"github.com/SergioLacerda/strategist-skill/internal/compile",
		"github.com/SergioLacerda/strategist-skill/internal/install",
		"github.com/SergioLacerda/strategist-skill/internal/stale",
		"github.com/SergioLacerda/strategist-skill/internal/embed",
	}

	deps := string(out)
	for _, pkg := range forbidden {
		if strings.Contains(deps, pkg) {
			t.Errorf("domain must not import %s — keep domain as a pure contract layer", pkg)
		}
	}
}

// TestLateralIsolation verifies that internal business-logic packages do not
// import each other. Each package depends only on internal/domain, never on a
// peer. Lateral coupling creates hidden coordination costs and circular-dep risk.
func TestLateralIsolation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		pkg       string
		forbidden []string
	}{
		{
			pkg: "github.com/SergioLacerda/strategist-skill/internal/compile",
			forbidden: []string{
				"github.com/SergioLacerda/strategist-skill/internal/install",
				"github.com/SergioLacerda/strategist-skill/internal/stale",
				"github.com/SergioLacerda/strategist-skill/internal/embed",
			},
		},
		{
			pkg: "github.com/SergioLacerda/strategist-skill/internal/install",
			forbidden: []string{
				"github.com/SergioLacerda/strategist-skill/internal/compile",
				"github.com/SergioLacerda/strategist-skill/internal/stale",
				"github.com/SergioLacerda/strategist-skill/internal/embed",
			},
		},
		{
			pkg: "github.com/SergioLacerda/strategist-skill/internal/stale",
			forbidden: []string{
				"github.com/SergioLacerda/strategist-skill/internal/compile",
				"github.com/SergioLacerda/strategist-skill/internal/install",
				"github.com/SergioLacerda/strategist-skill/internal/embed",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.pkg, func(t *testing.T) {
			t.Parallel()
			out, err := exec.Command("go", "list", "-deps", tc.pkg).CombinedOutput()
			if err != nil {
				t.Fatalf("go list -deps failed: %v\n%s", err, out)
			}
			deps := string(out)
			for _, forbidden := range tc.forbidden {
				if strings.Contains(deps, forbidden) {
					t.Errorf("%s must not import %s", tc.pkg, forbidden)
				}
			}
		})
	}
}
