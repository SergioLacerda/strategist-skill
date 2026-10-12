package runbook

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/spf13/cobra"
)

func TestRegisterAddsOnlyRunbookCommand(t *testing.T) {
	root := &cobra.Command{Use: "strategist"}
	Register(root)
	if _, _, err := root.Find([]string{"runbook", "select"}); err != nil {
		t.Fatal(err)
	}
}

func TestResolveRootUsesExplicitAndWorkspaceRoots(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), ".strategist")
	assertExplicitRoot(t, explicit)
	assertWorkspaceRoot(t, t.TempDir())
}

func assertExplicitRoot(t *testing.T, explicit string) {
	t.Helper()
	got, err := resolveRoot(explicit)
	if err != nil || got != explicit {
		t.Fatalf("explicit root: %q %v", got, err)
	}
}

func assertWorkspaceRoot(t *testing.T, workspace string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if got, err := resolveRoot(""); err != nil || got != workspace {
		t.Fatalf("workspace root: %q %v", got, err)
	}
	if err := os.Mkdir(filepath.Join(workspace, ".strategist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveRoot(""); err != nil || got != filepath.Join(workspace, ".strategist") {
		t.Fatalf("strategist root: %q %v", got, err)
	}
}

func TestNewProviderFailsClosedWithoutCatalog(t *testing.T) {
	if _, err := newProvider(t.TempDir(), "/bin/treasure-chest-tmp"); err == nil || !strings.Contains(err.Error(), "read ROSTER catalog") {
		t.Fatalf("unexpected error: %v", err)
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte("not: [catalog"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := newProvider(root, "/bin/treasure-chest-tmp"); err == nil || !strings.Contains(err.Error(), "parse ROSTER catalog") {
		t.Fatalf("unexpected parse error: %v", err)
	}
}

func TestNewProviderResolvesAdmittedWeaponAndRunSelectUsesIt(t *testing.T) {
	root := t.TempDir()
	writeProviderCatalog(t, root)
	providerScript := writeProviderScript(t, root)
	if _, err := newProvider(root, providerScript); err != nil {
		t.Fatal(err)
	}
	t.Setenv(providerExecutableEnv, providerScript)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringArray("signal", []string{"fix"}, "signal")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := runSelect(cmd, options{root: root, requester: "ranger", mission: "m-1", format: "json"}, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "fix") {
		t.Fatalf("selection output: %s", output.String())
	}
	if err := runSelect(cmd, options{root: root, executable: providerScript, requester: "sniper", mission: "m-1", format: "json"}, nil); err == nil || !strings.Contains(err.Error(), "requester") {
		t.Fatalf("unexpected requester error: %v", err)
	}
}

func writeProviderCatalog(t *testing.T, root string) {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	plugins := filepath.Join(root, "plugins")
	if err := os.MkdirAll(plugins, 0o755); err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "../../.."))
	catalog, err := os.ReadFile(filepath.Join(repoRoot, "internal/embed/defaults/plugins/catalog.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plugins, "catalog.yaml"), catalog, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeProviderScript(t *testing.T, root string) string {
	t.Helper()
	providerScript := filepath.Join(root, "provider.sh")
	script := "#!/bin/sh\nif [ \"$1\" = status ]; then echo '{\"Provider\":\"treasure-chest-tmp\",\"Version\":\"0.1.0\",\"SchemaVersion\":1,\"Healthy\":true}'; else echo '{\"Envelope\":{\"Provider\":\"treasure-chest-tmp\",\"Version\":\"0.1.0\",\"SchemaVersion\":1,\"Freshness\":\"fresh\",\"FallbackState\":\"none\"},\"Items\":[{\"ID\":\"fix\",\"Kind\":\"potion\",\"Score\":1,\"Trust\":1,\"Freshness\":\"fresh\",\"Applicability\":\"accepted\",\"Reason\":\"matched\",\"Source\":{\"Path\":\"docs/runbooks/fix.md\"}}]}'; fi\n"
	if err := os.WriteFile(providerScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return providerScript
}

func TestRunSelectRejectsMissingSignalAndBinding(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("signal", nil, "signal")
	if err := runSelect(cmd, options{}, nil); err == nil || !strings.Contains(err.Error(), "at least one --signal") {
		t.Fatalf("unexpected signal error: %v", err)
	}
	cmd.Flags().Set("signal", "fix")
	if err := runSelect(cmd, options{root: t.TempDir()}, nil); err == nil || !strings.Contains(err.Error(), "provider binding") {
		t.Fatalf("unexpected binding error: %v", err)
	}
	if err := runSelect(cmd, options{root: t.TempDir(), executable: "/bin/false"}, nil); err == nil || !strings.Contains(err.Error(), "mission identifier") {
		t.Fatalf("unexpected mission error: %v", err)
	}
}

func TestRenderJSONAndTable(t *testing.T) {
	projection := jeweler.Projection{Items: []jeweler.Item{{ID: "potion", Source: jeweler.Source{Path: "docs/runbooks/fix.md"}, Reason: "matched"}}}
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := render(cmd, projection, "json"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "fix") {
		t.Fatalf("json output: %s", out.String())
	}
	out.Reset()
	if err := render(cmd, projection, "table"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "RUNBOOK_ID") {
		t.Fatalf("table output: %s", out.String())
	}
	if err := render(cmd, jeweler.Projection{}, "table"); err != nil {
		t.Fatal(err)
	}
	if err := render(cmd, projection, "yaml"); err == nil {
		t.Fatal("expected format error")
	}
	if err := render(cmd, jeweler.Projection{Items: []jeweler.Item{{ID: "bad"}}}, "table"); err == nil {
		t.Fatal("expected source error")
	}
}

type runbookFailWriter struct{}

func (runbookFailWriter) Write([]byte) (int, error) { return 0, os.ErrClosed }

func TestRenderJSONReportsOutputFailure(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(runbookFailWriter{})
	projection := jeweler.Projection{Items: []jeweler.Item{{ID: "potion", Source: jeweler.Source{Path: "fix.md"}}}}
	if err := render(cmd, projection, "json"); err == nil || !strings.Contains(err.Error(), "render runbook selection JSON") {
		t.Fatalf("unexpected output error: %v", err)
	}
	if err := render(cmd, projection, "table"); err == nil || !strings.Contains(err.Error(), "render runbook selection header") {
		t.Fatalf("unexpected table output error: %v", err)
	}
}

func TestRunbookIDFallsBackToSourceBase(t *testing.T) {
	if got := runbookID(jeweler.Item{ID: "fallback", Source: jeweler.Source{Path: ""}}); got != "fallback" {
		t.Fatalf("empty path id: %q", got)
	}
	if got := runbookID(jeweler.Item{ID: "fallback", Source: jeweler.Source{Path: "docs/runbooks/a.md"}}); got != "a" {
		t.Fatalf("path id: %q", got)
	}
}
