package mission

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

type workspaceContextReader struct{ root string }

func (r workspaceContextReader) ReadFile(ref string) ([]byte, error) {
	raw, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(ref))) //nolint:gosec // domain validates ref as relative
	if err != nil {
		return nil, fmt.Errorf("read context reference %q: %w", ref, err)
	}
	return raw, nil
}

// NewContext builds `mission context`.
func NewContext(deps LifecycleDependencies) *cobra.Command {
	var refs, digests []string
	maxRefs, maxBytes := domain.DefaultContextMaxReferences, domain.DefaultContextMaxBytes
	cmd := &cobra.Command{Use: "context", Short: "Materialize declared mission context"}
	lf := bindLifecycleFlags(cmd, deps)
	f := cmd.Flags()
	f.StringSliceVar(&refs, "ref", nil, "declared workspace-relative context reference (repeatable)")
	f.StringSliceVar(&digests, "digest", nil, "expected sha256 digest for each --ref, in the same order")
	f.IntVar(&maxRefs, "max-refs", maxRefs, "maximum number of references")
	f.IntVar(&maxBytes, "max-bytes", maxBytes, "maximum materialized bytes")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunContext(cmd, deps, lf.root, lf.missionID, lf.asJSON, refs, digests, maxRefs, maxBytes)
	}
	return cmd
}

// RunContext materializes the declared workspace references of a known
// mission through domain.MaterializeContext.
func RunContext(cmd *cobra.Command, deps LifecycleDependencies, rootInput, missionID string, asJSON bool, refs, digests []string, maxRefs, maxBytes int) error {
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	root, _, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission context: %w", err)
	}
	contextRefs := make([]domain.ContextReference, len(refs))
	for i, ref := range refs {
		contextRefs[i] = domain.ContextReference{Ref: ref, Kind: "context"}
		if i < len(digests) {
			contextRefs[i].Digest = digests[i]
		}
	}
	materialized, err := application.MaterializeMissionContext(
		deps.Load, root, missionID, workspaceContextReader{root: filepath.Dir(root)}, contextRefs, maxRefs, maxBytes,
	)
	if err != nil {
		return fmt.Errorf("mission context: %w", err)
	}
	return deps.WriteResult(cmd, asJSON, materialized)
}
