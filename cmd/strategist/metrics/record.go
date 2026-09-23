package metrics

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// RecordOptions holds the flags of `strategist metrics record`.
type RecordOptions struct {
	Root, Mission, Run, Agent, ClaimFile, CorrelationKey, Reason string
	Missing                                                      bool
}
type confidenceClaimFile struct {
	Claim    domain.ConfidenceClaim `yaml:"claim"`
	Evidence []domain.Evidence      `yaml:"evidence"`
}

// NewRecord builds the `metrics record` command.
func NewRecord(deps Dependencies) *cobra.Command {
	opts := RecordOptions{}
	cmd := &cobra.Command{Use: "record", Short: "Record a confidence claim or an explicit missing-record for a boundary"}
	f := cmd.Flags()
	f.StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.Mission, "mission", "", "mission id (required)")
	f.StringVar(&opts.Run, "run", "", "optional explicit run id for a repeated role execution")
	f.StringVar(&opts.Agent, "agent", "", "producing agent (required)")
	f.StringVar(&opts.ClaimFile, "claim-file", "", `YAML file with the claim and its evidence; "-" reads it from standard input (preferred: no file is created), and a file inside the workspace base_path is rejected`)
	f.BoolVar(&opts.Missing, "missing", false, "record an explicit missing-record instead of a claim")
	f.StringVar(&opts.CorrelationKey, "correlation-key", "", "boundary correlation key (with --missing)")
	f.StringVar(&opts.Reason, "reason", "", "why no summary was produced (with --missing)")
	for _, name := range []string{"mission", "agent"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunRecord(cmd, deps, opts) }
	return cmd
}

// RunRecord records one confidence claim, or an explicit missing-record, for
// a mission boundary.
func RunRecord(cmd *cobra.Command, deps Dependencies, opts RecordOptions) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	root, err := deps.ResolveRoot(cmd, "record", opts.Root)
	if err != nil {
		return err
	}
	producer, err := telemetry.NewConfidenceProducerAdapter(telemetry.ConfidenceHistoryPath(root), opts.Agent, opts.Mission)
	if err != nil {
		return fmt.Errorf("metrics record: %w", err)
	}
	producer = producer.WithRun(opts.Run)
	if opts.Missing {
		return recordMissing(cmd, producer, opts)
	}
	return recordClaim(cmd, deps, root, producer, opts.ClaimFile)
}
func recordMissing(cmd *cobra.Command, producer telemetry.ConfidenceProducerAdapter, opts RecordOptions) error {
	if opts.CorrelationKey == "" || opts.Reason == "" {
		return fmt.Errorf("metrics record: --missing requires --correlation-key and --reason")
	}
	if err := producer.RecordMissing(opts.CorrelationKey, opts.Reason); err != nil {
		return fmt.Errorf("metrics record: %w", err)
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "missing-record recorded: agent=%s mission=%s correlation_key=%s\n", producer.Agent, producer.MissionID, opts.CorrelationKey)
	if err != nil {
		return fmt.Errorf("metrics record: write output: %w", err)
	}
	return nil
}
func recordClaim(cmd *cobra.Command, deps Dependencies, root string, producer telemetry.ConfidenceProducerAdapter, path string) error {
	if path == "" {
		return fmt.Errorf("metrics record: --claim-file is required unless --missing is set (use `--claim-file -` to read from standard input)")
	}
	input, err := readClaimInput(cmd, deps, root, path)
	if err != nil {
		return err
	}
	record, err := producer.RecordClaim(input.Claim, input.Evidence)
	if err != nil {
		return fmt.Errorf("metrics record: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "coverage_status: %s\nclaim: %s\nviolation: %s\n", record.CoverageStatus, record.ClaimID, record.Violation); err != nil {
		return fmt.Errorf("metrics record: write output: %w", err)
	}
	// The rejected observation is persisted for the confidence metrics, but the
	// command still fails: an exit status of 0 let rejections go unnoticed.
	if record.CoverageStatus == telemetry.ConfidenceCoverageRejected {
		return fmt.Errorf("metrics record: claim %s rejected (recorded as rejected): %s", record.ClaimID, record.Violation)
	}
	return nil
}

// claimFileShape is shown whenever a claim file does not have the expected
// top-level layout.
const claimFileShape = "expected top-level keys `claim:` (the claim mapping) and optional `evidence:` (a list)"

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

// decodeClaim decodes a claim document, rejecting a missing `claim` mapping
// and unknown top-level keys before normalization. A flat file used to decode
// silently into an empty claim and fail later with an unrelated message.
func decodeClaim(raw []byte) (confidenceClaimFile, error) {
	var top map[string]yaml.Node
	if err := yaml.Unmarshal(raw, &top); err != nil {
		return confidenceClaimFile{}, fmt.Errorf("metrics record: parse claim file: %w", err)
	}
	if err := checkClaimFileKeys(top); err != nil {
		return confidenceClaimFile{}, err
	}
	var input confidenceClaimFile
	if err := yaml.Unmarshal(raw, &input); err != nil {
		return confidenceClaimFile{}, fmt.Errorf("metrics record: parse claim file: %w", err)
	}
	return input, nil
}

func checkClaimFileKeys(top map[string]yaml.Node) error {
	for key := range top {
		if key != "claim" && key != "evidence" {
			return fmt.Errorf("metrics record: claim file has unknown top-level key %q; %s", key, claimFileShape)
		}
	}
	if claim, ok := top["claim"]; !ok || claim.Kind != yaml.MappingNode {
		return fmt.Errorf("metrics record: claim file has no `claim` mapping; %s", claimFileShape)
	}
	return nil
}
