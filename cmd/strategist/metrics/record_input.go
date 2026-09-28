package metrics

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// confidenceClaimFile is one claim document: either a single `claim:` or a
// batch `claims:` list, both with the evidence they cite.
type confidenceClaimFile struct {
	Claim    domain.ConfidenceClaim   `yaml:"claim"`
	Claims   []domain.ConfidenceClaim `yaml:"claims"`
	Evidence []domain.Evidence        `yaml:"evidence"`
	batch    bool
}

// claimFileShape is shown whenever a claim file does not have the expected
// top-level layout.
const claimFileShape = "expected top-level keys `claim:` (the claim mapping) or `claims:` (a list of claim mappings), and optional `evidence:` (a list)"

// claimFromStdin is the --claim-file value that reads the claim from standard
// input instead of a file, so recording confidence creates no file at all.
const claimFromStdin = "-"

// readClaimInput loads the claim from standard input or from a file. A file
// inside the workspace base_path is rejected before it is read: the claim is
// transient CLI input and must not become a workspace artifact.
func readClaimInput(cmd *cobra.Command, deps Dependencies, root, path string) (confidenceClaimFile, error) {
	if path == claimFromStdin {
		raw, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return confidenceClaimFile{}, fmt.Errorf("metrics record: read standard input: %w", err)
		}
		if strings.TrimSpace(string(raw)) == "" {
			return confidenceClaimFile{}, fmt.Errorf("metrics record: standard input is empty; pipe the claim document into `--claim-file -`")
		}
		return decodeClaim(raw)
	}
	if err := rejectClaimUnderBasePath(deps, root, path); err != nil {
		return confidenceClaimFile{}, err
	}
	raw, err := os.ReadFile(path) //nolint:gosec // path is an operator-supplied claim file
	if err != nil {
		return confidenceClaimFile{}, fmt.Errorf("metrics record: read claim file: %w", err)
	}
	return decodeClaim(raw)
}

// rejectClaimUnderBasePath fails closed when the claim file resolves (through
// `..` segments and symbolic links) to a location inside the workspace
// base_path. It is skipped only when no resolver is wired or the workspace
// declares no base_path.
func rejectClaimUnderBasePath(deps Dependencies, root, path string) error {
	if deps.ResolveBasePath == nil {
		return nil
	}
	base, err := deps.ResolveBasePath(root)
	if err != nil {
		return fmt.Errorf("metrics record: resolve base_path to check the claim file location: %w", err)
	}
	if base == "" {
		return nil
	}
	rel, err := filepath.Rel(realPath(base), realPath(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil
	}
	return fmt.Errorf("metrics record: claim file %s is inside the workspace artifact tree (%s); the claim is transient input and must not be stored there. Pipe it instead: `strategist metrics record ... --claim-file - <<'EOF'`", path, base)
}

// realPath returns an absolute path with symbolic links evaluated, falling
// back to the cleaned absolute path when the target does not exist yet.
func realPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// decodeClaim decodes a claim document, rejecting a missing `claim`/`claims`
// mapping and unknown top-level keys before normalization. A flat file used to
// decode silently into an empty claim and fail later with an unrelated message.
func decodeClaim(raw []byte) (confidenceClaimFile, error) {
	var top map[string]yaml.Node
	if err := yaml.Unmarshal(raw, &top); err != nil {
		return confidenceClaimFile{}, fmt.Errorf("metrics record: parse claim file: %w", err)
	}
	batch, err := checkClaimFileKeys(top)
	if err != nil {
		return confidenceClaimFile{}, err
	}
	var input confidenceClaimFile
	if err := yaml.Unmarshal(raw, &input); err != nil {
		return confidenceClaimFile{}, fmt.Errorf("metrics record: parse claim file: %w", err)
	}
	input.batch = batch
	return input, nil
}

// checkClaimFileKeys validates the top-level layout and reports whether the
// document is a batch (`claims:`) rather than a single `claim:`.
func checkClaimFileKeys(top map[string]yaml.Node) (bool, error) {
	for key := range top {
		if key != "claim" && key != "claims" && key != "evidence" {
			return false, fmt.Errorf("metrics record: claim file has unknown top-level key %q; %s", key, claimFileShape)
		}
	}
	claim, hasClaim := top["claim"]
	claims, hasClaims := top["claims"]
	switch {
	case hasClaim && hasClaims:
		return false, fmt.Errorf("metrics record: claim file declares both `claim:` and `claims:`; use either `claim:` or `claims:`, not both")
	case hasClaims:
		return true, checkClaimsList(claims)
	case !hasClaim || claim.Kind != yaml.MappingNode:
		return false, fmt.Errorf("metrics record: claim file has no `claim` mapping; %s", claimFileShape)
	}
	return false, nil
}

func checkClaimsList(node yaml.Node) error {
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("metrics record: `claims` must be a list of claim mappings; %s", claimFileShape)
	}
	if len(node.Content) == 0 {
		return fmt.Errorf("metrics record: the `claims` list is empty; nothing to record")
	}
	return nil
}
