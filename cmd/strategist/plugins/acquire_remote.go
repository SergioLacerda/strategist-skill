package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/plugins/acquisition"
	"github.com/spf13/cobra"
)

// NewAcquireRemote creates the explicit, quarantine-only remote acquisition
// command. It never calls prepare-embedded or promotes a package.
func NewAcquireRemote() *cobra.Command {
	request := acquisition.Request{Source: acquisition.SourceGit}
	cmd := &cobra.Command{
		Use:   "acquire-remote",
		Short: "Acquire a pinned Git Skill-for-Hire package into quarantine",
		Long: `Fetches one full-commit Git reference into an isolated temporary
quarantine, validates the package shape and writes acquisition.json plus READY.
It never writes external-skills-source, .strategist, the embedded catalog or a
lockfile. Promotion remains a separate, reviewed operator action.
Network access is opt-in and tests should omit --allow-network.`,
	}
	cmd.Flags().StringVar(&request.Source, "source", acquisition.SourceGit, "remote source class (only git is supported)")
	cmd.Flags().StringVar(&request.Repository, "repository", "", "Git repository URI or absolute local fixture path (required)")
	cmd.Flags().StringVar(&request.Commit, "commit", "", "full 40-character lowercase commit SHA (required)")
	cmd.Flags().StringVar(&request.PackageID, "package-id", "", "package ID expected for the imported skill (required)")
	cmd.Flags().BoolVar(&request.AllowNetwork, "allow-network", false, "explicitly authorize the Git fetch")
	cmd.Flags().StringVar(&request.QuarantineRoot, "quarantine-root", "", "optional quarantine parent; must be outside the workspace")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunAcquireRemote(cmd.Context(), cmd.OutOrStdout(), request)
	}
	return cmd
}

// RunAcquireRemote adapts the CLI request to the isolated acquisition core.
func RunAcquireRemote(ctx context.Context, out io.Writer, request acquisition.Request) error {
	if request.QuarantineRoot == "" {
		request.QuarantineRoot = os.TempDir()
	}
	workspace, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("acquire-remote: resolve workspace: %w", err)
	}
	request.ForbiddenRoots = append(request.ForbiddenRoots,
		workspace,
		filepath.Join(workspace, ".strategist"),
		filepath.Join(workspace, "external-skills-source"),
	)
	result, err := acquisition.Acquire(ctx, request)
	if err != nil {
		return fmt.Errorf("acquire-remote: %w", err)
	}
	if _, err := fmt.Fprintf(out, "acquire-remote: quarantined %s\n", result.QuarantinePath); err != nil {
		return fmt.Errorf("acquire-remote: write report: %w", err)
	}
	if _, err := fmt.Fprintf(out, "acquire-remote: commit=%s package=%s tree_digest=%s\n", result.Manifest.Commit, result.Manifest.PackageID, result.Manifest.TreeDigest); err != nil {
		return fmt.Errorf("acquire-remote: write report: %w", err)
	}
	return nil
}
