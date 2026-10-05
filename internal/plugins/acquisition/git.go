package acquisition

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRunner is the narrow boundary between acquisition policy and Git.
type GitRunner interface {
	FetchArchive(ctx context.Context, workDir, repository, commit string, allowNetwork bool) ([]byte, string, error)
}

// CommandRunner executes one argument-vector command without a shell.
type CommandRunner interface {
	Run(ctx context.Context, workDir string, args ...string) ([]byte, error)
}

// CommandGitRunner is the production Git implementation. It never resolves a
// branch or tag: the fetch refspec is the validated full commit SHA.
type CommandGitRunner struct {
	Command CommandRunner
}

// FetchArchive acquires and archives exactly one validated commit.
func (g CommandGitRunner) FetchArchive(ctx context.Context, workDir, repository, commit string, allowNetwork bool) ([]byte, string, error) {
	if !allowNetwork {
		return nil, "", acquisitionError("network_opt_in_required", "pass --allow-network explicitly")
	}
	runner := g.Command
	if runner == nil {
		runner = OSCommandRunner{}
	}
	gitWorkDir, err := createGitWorkDir(workDir)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		if err := os.RemoveAll(gitWorkDir); err != nil {
			return
		}
	}() //nolint:gosec // exact transaction child created above
	if err := initializeGitWorkDir(ctx, runner, gitWorkDir, repository, commit); err != nil {
		return nil, "", err
	}
	return archiveResolvedCommit(ctx, runner, gitWorkDir)
}

func createGitWorkDir(workDir string) (string, error) {
	gitWorkDir := filepath.Join(workDir, ".git-work")
	if err := os.MkdirAll(gitWorkDir, 0o700); err != nil {
		return "", acquisitionError("git_workspace_failed", err.Error())
	}
	return gitWorkDir, nil
}

func initializeGitWorkDir(ctx context.Context, runner CommandRunner, workDir, repository, commit string) error {
	commands := [][]string{
		{"init", "--quiet"},
		{"remote", "add", "origin", repository},
		{"fetch", "--no-tags", "--depth=1", "origin", commit},
	}
	for _, args := range commands {
		if _, err := runner.Run(ctx, workDir, args...); err != nil {
			return acquisitionError("git_command_failed", err.Error())
		}
	}
	return nil
}

func archiveResolvedCommit(ctx context.Context, runner CommandRunner, workDir string) ([]byte, string, error) {
	resolvedRaw, err := runner.Run(ctx, workDir, "rev-parse", "FETCH_HEAD^{commit}")
	if err != nil {
		return nil, "", acquisitionError("git_command_failed", err.Error())
	}
	resolved := strings.TrimSpace(string(resolvedRaw))
	if err := ValidateCommit(resolved); err != nil {
		return nil, "", acquisitionError("commit_resolution_invalid", err.Error())
	}
	archive, err := runner.Run(ctx, workDir, "archive", "--format=tar", resolved)
	if err != nil {
		return nil, "", acquisitionError("git_command_failed", err.Error())
	}
	return archive, resolved, nil
}

// OSCommandRunner provides the non-shell execution used in production.
type OSCommandRunner struct{}

// Run executes Git with an argument vector and no shell interpolation.
func (OSCommandRunner) Run(ctx context.Context, workDir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // args are validated and passed without a shell
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git command: %w", err)
	}
	return output, nil
}
