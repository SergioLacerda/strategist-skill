package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

// claudeArgs lists the Claude flags. The prompt travels on stdin because
// --tools is variadic and would swallow a trailing positional prompt. --bare
// disables OAuth and keychain auth, so it is used only with an API key.
func claudeArgs(hasAPIKey bool) []string {
	args := []string{"--print"}
	if hasAPIKey {
		args = append(args, "--bare")
	}
	return append(args, "--permission-mode", "plan", "--no-session-persistence", "--tools=Read,Glob,Grep")
}

func runClaudePrompt(ctx context.Context, workspace, prompt string) (string, error) {
	return runClaudeBinary(ctx, "claude", workspace, prompt, maxHostResultBytes)
}

func runClaudeBinary(ctx context.Context, binary, workspace, prompt string, limit int) (string, error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	//nolint:gosec // G204: Claude is a fixed executable selected by the closed host switch.
	cmd := exec.CommandContext(runCtx, binary, claudeArgs(os.Getenv("ANTHROPIC_API_KEY") != "")...)
	cmd.Dir = workspace
	cmd.Stdin = strings.NewReader(prompt)
	streams := attachHostStreams(cmd, cancel, limit)
	if err := cmd.Run(); err != nil {
		return "", streams.failure("claude", err)
	}
	return requireHostResult(streams.stdout.buf.Bytes())
}
