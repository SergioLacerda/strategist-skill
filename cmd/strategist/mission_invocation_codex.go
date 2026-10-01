package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func runCodexPrompt(ctx context.Context, workspace, prompt string) (string, error) {
	stateDir, err := os.MkdirTemp("", "strategist-codex-state-*")
	if err != nil {
		return "", fmt.Errorf("create Codex transient state directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stateDir) }() //nolint:errcheck // best-effort cleanup of a private transient state directory.

	outputPath, err := createCodexOutputFile(stateDir)
	if err != nil {
		return "", err
	}
	//nolint:gosec // G204: Codex is a fixed executable selected by the closed host switch.
	cmd := exec.CommandContext(ctx, "codex", codexExecArgs(stateDir, outputPath, workspace)...)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Dir = workspace
	cmd.Env = codexBridgeEnv(os.Environ(), stateDir)
	if err := linkCodexAuth(os.Environ(), stateDir); err != nil {
		return "", err
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("codex host bridge: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if info, err := os.Stat(outputPath); err == nil && info.Size() > maxHostResultBytes {
		return "", fmt.Errorf("codex host bridge: %w", errHostOutputTooLarge)
	}
	result, err := os.ReadFile(outputPath) //nolint:gosec // G304: outputPath is created above with os.CreateTemp.
	if err != nil {
		return "", fmt.Errorf("read Codex host result: %w", err)
	}
	return requireHostResult(result)
}

func createCodexOutputFile(stateDir string) (string, error) {
	output, err := os.CreateTemp(stateDir, "result-*.md")
	if err != nil {
		return "", fmt.Errorf("create Codex output file: %w", err)
	}
	outputPath := output.Name()
	if err := output.Close(); err != nil {
		return "", fmt.Errorf("close Codex output file: %w", err)
	}
	return outputPath, nil
}

// codexExecArgs ends with "-" so Codex reads the prompt from stdin, keeping
// the untrusted request out of the process list and under ARG_MAX.
func codexExecArgs(stateDir, outputPath, workspace string) []string {
	return []string{
		"exec",
		"--ephemeral",
		"--ignore-rules",
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"--config", "sqlite_home=" + strconv.Quote(stateDir),
		"--config", "log_dir=" + strconv.Quote(filepath.Join(stateDir, "log")),
		"--output-last-message", outputPath,
		"--cd", workspace,
		"-",
	}
}

// linkCodexAuth exposes only the user's Codex credentials inside the private
// CODEX_HOME so the bridge stays authenticated without sharing other state.
// A missing auth.json is not an error: Codex may authenticate another way.
func linkCodexAuth(environment []string, stateDir string) error {
	source := filepath.Join(codexHomeDir(environment), "auth.json")
	if _, err := os.Stat(source); err != nil {
		return nil //nolint:nilerr // absent credentials are tolerated.
	}
	if err := os.Symlink(source, filepath.Join(stateDir, "auth.json")); err != nil {
		return fmt.Errorf("link Codex credentials: %w", err)
	}
	return nil
}

func codexHomeDir(environment []string) string {
	for _, entry := range environment {
		if value, ok := strings.CutPrefix(entry, "CODEX_HOME="); ok && value != "" {
			return value
		}
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(userHome, ".codex")
}

func codexBridgeEnv(environment []string, stateDir string) []string {
	const codexHomePrefix = "CODEX_HOME="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, codexHomePrefix) {
			result = append(result, entry)
		}
	}
	return append(result, codexHomePrefix+stateDir)
}
