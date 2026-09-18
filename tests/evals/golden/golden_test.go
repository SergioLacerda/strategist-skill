//go:build golden

// Kept behind the golden tag (like eval/spec/integration) rather than in the
// default `go test ./...` path: the cli-help subtest alone costs ~90s (a cold
// `go run ./cmd/strategist --help` per invocation), which would otherwise tax
// every plain test run. See docs/adr/0026-deterministic-golden-testing.md.
package golden

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
)

func TestDeterministicArtifacts(t *testing.T) {
	root := repositoryRoot(t)
	tests := []struct {
		name   string
		golden string
		mode   Mode
		load   func(*testing.T) []byte
	}{
		{"handoff-manifest", "handoffs/archivist-to-sniper.json", Normalized, read(root, "internal/embed/defaults/schemas/handoff-archivist-to-sniper.schema.yaml")},
		{"provider-manifest", "manifests/brainstorming.json", Normalized, read(root, "internal/embed/defaults/skills/brainstorming/skill.yaml")},
		{"telemetry-attributes", "telemetry/attribute-keys.txt", Exact, telemetryAttributes(root)},
		{"cli-help", "cli/help.txt", Exact, cliHelp(root)},
		{"rendered-schema", "schemas/intake.json", Normalized, read(root, "internal/embed/defaults/schemas/intake.schema.yaml")},
		{"compiled-contract-shape", "compiled/domain-shape.json", Structural, read(root, "internal/embed/defaults/contracts/machine/compile-domain.yaml")},
		{"default-config", "defaults/roles.json", Normalized, read(root, "internal/embed/defaults/roles/default.yaml")},
		{"compiled-prompt", "prompts/epic-standalone.txt", Exact, read(root, "internal/embed/defaults/templates/epic-standalone.yaml")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join("testdata", test.golden)
			if err := Assert(path, test.load(t), test.mode); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCanonicalizeVolatileFields(t *testing.T) {
	input := []byte(`timestamp: 2026-08-11T12:34:56Z
request_uuid: 550e8400-e29b-41d4-a716-446655440000
artifact_path: C:\\Users\\runner\\AppData\\Local\\Temp\\artifact.yaml
checksum: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
duration: 125ms
hostname: build-agent-17
stable: retained
`)
	got, err := Canonicalize(input, Normalized)
	if err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range []string{"<timestamp>", "<uuid>", "<temp-path>", "<hash>", "<duration>", "<hostname>", "retained"} {
		if !strings.Contains(string(got), sentinel) {
			t.Errorf("canonical output does not contain %q:\n%s", sentinel, got)
		}
	}
}

func TestGoldenProvenanceCatalog(t *testing.T) {
	data := readFile(t, "goldens.yaml")
	var catalog struct {
		Golden []struct {
			ID        string   `yaml:"id"`
			Source    string   `yaml:"source"`
			Mode      string   `yaml:"mode"`
			Contracts []string `yaml:"contracts"`
			Reason    string   `yaml:"update_reason"`
		} `yaml:"golden"`
	}
	if err := yamlUnmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Golden) != 8 {
		t.Fatalf("got %d provenance entries, want 8", len(catalog.Golden))
	}
	seen := map[string]bool{}
	for _, item := range catalog.Golden {
		if item.ID == "" || item.Source == "" || item.Mode == "" || item.Reason == "" || len(item.Contracts) == 0 || seen[item.ID] {
			t.Fatalf("invalid provenance entry: %#v", item)
		}
		if item.Mode != string(Exact) && item.Mode != string(Normalized) && item.Mode != string(Structural) {
			t.Fatalf("invalid comparison mode for %s: %s", item.ID, item.Mode)
		}
		seen[item.ID] = true
	}
}

func TestAuthorityFixturesRemainIndependent(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		authority string
		bridge    string
		connector string
	}{
		{"standalone", "authority/standalone.json", "standalone", "absent", "native"},
		{"bridge-nil", "authority/governance-bridge-nil.json", "governance_bridge_nil", "nil", "native"},
		{"bridge-read-only", "authority/governance-bridge-read-only.json", "governance_bridge_read_only", "read_only", "native"},
		{"connector-unsupported", "authority/connector-unsupported.json", "connector_unsupported", "absent", "unsupported"},
	}
	seen := map[string]bool{}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			data := readFile(t, filepath.Join("testdata", test.path))
			var fixture struct {
				Authority           string `yaml:"authority"`
				Bridge              string `yaml:"governance_bridge"`
				Connector           string `yaml:"connector"`
				LiveProviderInvoked bool   `yaml:"live_provider_invoked"`
			}
			if err := yamlUnmarshal(data, &fixture); err != nil {
				t.Fatal(err)
			}
			if seen[fixture.Authority] || fixture.Authority != test.authority || fixture.Bridge != test.bridge || fixture.Connector != test.connector || fixture.LiveProviderInvoked {
				t.Fatalf("authority fixture conflates evidence boundaries: %#v", fixture)
			}
			seen[fixture.Authority] = true
		})
	}
}

func TestEmbeddedDefaultsMatchCanonicalSources(t *testing.T) {
	root := repositoryRoot(t)
	runtimeRoot := t.TempDir()
	if err := (embedpkg.Extractor{}).Extract(runtimeRoot, true); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"schemas/handoff-archivist-to-sniper.schema.yaml",
		"skills/brainstorming/skill.yaml",
		"templates/epic-standalone.yaml",
	} {
		t.Run(rel, func(t *testing.T) {
			source := readFile(t, filepath.Join(root, "internal", "embed", "defaults", rel))
			runtime := readFile(t, filepath.Join(runtimeRoot, rel))
			if !bytes.Equal(source, runtime) {
				t.Fatalf("embedded materialization drifted from canonical source: %s", rel)
			}
		})
	}
}

func TestCanonicalizeStructuredValuesDeterministically(t *testing.T) {
	input := []byte("items:\n  - name: alpha\n    count: 2\n  - name: alpha\n    count: 9\nmeta:\n  stable: true\n")
	got, err := Canonicalize(input, Structural)
	if err != nil {
		t.Fatal(err)
	}
	want := "\"items\": [\n    {\n      \"count\": \"number\",\n      \"name\": \"string\"\n    }\n  ]"
	if !strings.Contains(string(got), want) {
		t.Fatalf("structural canonicalization missing stable shape:\n%s", got)
	}
}

func TestCanonicalizeRejectsUnknownMode(t *testing.T) {
	_, err := Canonicalize([]byte("value: true\n"), Mode("unknown"))
	if err == nil || !strings.Contains(err.Error(), "unknown comparison mode") {
		t.Fatalf("expected unknown mode error, got %v", err)
	}
}

func TestAssertReportsDriftWithoutRewritingGolden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.txt")
	want := []byte("expected\n")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	err := Assert(path, []byte("actual\n"), Exact)
	if err == nil || !strings.Contains(err.Error(), "artifact drift detected") || !strings.Contains(err.Error(), path) {
		t.Fatalf("expected actionable drift, got %v", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("drift check rewrote golden: %q", got)
	}
}

func TestAssertRejectsUpdateInCI(t *testing.T) {
	previousUpdate := *update
	*update = true
	t.Cleanup(func() { *update = previousUpdate })
	previousCI := os.Getenv("CI")
	if err := os.Setenv("CI", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Setenv("CI", previousCI) })

	err := Assert(filepath.Join(t.TempDir(), "fixture.txt"), []byte("value\n"), Exact)
	if err == nil || !strings.Contains(err.Error(), "-update is forbidden in CI") {
		t.Fatalf("expected CI update rejection, got %v", err)
	}
}

func read(root, path string) func(*testing.T) []byte {
	return func(t *testing.T) []byte { return readFile(t, filepath.Join(root, filepath.FromSlash(path))) }
}

func telemetryAttributes(root string) func(*testing.T) []byte {
	return func(t *testing.T) []byte {
		data := readFile(t, filepath.Join(root, "internal", "telemetry", "schema.go"))
		matches := regexp.MustCompile(`"strategist\.[^"]+"`).FindAll(data, -1)
		values := make([]string, 0, len(matches))
		for _, match := range matches {
			var value string
			if err := json.Unmarshal(match, &value); err != nil {
				t.Fatal(err)
			}
			values = append(values, value)
		}
		sort.Strings(values)
		return []byte(strings.Join(values, "\n"))
	}
}

func cliHelp(root string) func(*testing.T) []byte {
	return func(t *testing.T) []byte {
		cmd := exec.Command("go", "run", "./cmd/strategist", "--help")
		cmd.Dir = root
		// A nested `go run` must not share the parent test process' caches: the
		// parent may still hold build locks and a hosted runner may expose its
		// module cache as read-only.
		cacheRoot := filepath.Join(root, ".tmp-gocache-cli")
		moduleCache := filepath.Join(root, ".tmp-gomodcache-cli")
		cmd.Env = append(os.Environ(), "NO_COLOR=1", "GOCACHE="+cacheRoot, "GOMODCACHE="+moduleCache)
		var output, diagnostics bytes.Buffer
		cmd.Stdout = &output
		cmd.Stderr = &diagnostics
		err := cmd.Run()
		if err != nil {
			t.Fatalf("strategist --help: %v\n%s", err, diagnostics.String())
		}
		return output.Bytes()
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
