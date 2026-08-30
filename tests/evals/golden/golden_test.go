//go:build golden

// Kept behind the golden tag (like eval/spec/integration) rather than in the
// default `go test ./...` path: the cli-help subtest alone costs ~90s (a cold
// `go run ./cmd/strategist --help` per invocation), which would otherwise tax
// every plain test run. See docs/adr/0026-deterministic-golden-testing.md.
package golden

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
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
			Contracts []string `yaml:"contracts"`
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
		if item.ID == "" || len(item.Contracts) == 0 || seen[item.ID] {
			t.Fatalf("invalid provenance entry: %#v", item)
		}
		seen[item.ID] = true
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
		// A nested `go run` must not share the parent test process' build cache:
		// the parent may still hold cache locks while this command is compiled.
		cmd.Env = append(os.Environ(), "NO_COLOR=1", "GOCACHE="+filepath.Join(root, ".tmp-gocache-cli"))
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("strategist --help: %v\n%s", err, output)
		}
		return output
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
