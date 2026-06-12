package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type SpawnOptions struct {
	Model           string
	ReasoningEffort string
	MaxTokens       int
}

func spawnQoderCli(ctx context.Context, prompt string, opts SpawnOptions, cm *ConfigManager) (io.ReadCloser, io.ReadCloser, error) {
	config := cm.Get()
	
	binaryName := "qodercli"
	tokenVar := "QODER_PERSONAL_ACCESS_TOKEN"
	if strings.ToLower(config.Backend) == "cn" {
		binaryName = "qoderclicn"
		tokenVar = "QODERCN_PERSONAL_ACCESS_TOKEN"
	}

	// Prepare arguments
	args := []string{"-p", "-f", "stream-json", "--dangerously-skip-permissions", "--permission-mode", "bypassPermissions"}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.ReasoningEffort != "" {
		args = append(args, "--reasoning-effort", opts.ReasoningEffort)
	}
	if opts.MaxTokens > 0 {
		args = append(args, "--max-output-tokens", fmt.Sprintf("%d", opts.MaxTokens))
	}

	cmdPath := binaryName
	if runtime.GOOS == "windows" {
		cmdPath = binaryName + ".cmd"
	}

	AddSystemLog(fmt.Sprintf("Spawning %s (model: %s)", cmdPath, opts.Model), "info", "spawn")

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	
	// Set environment
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", tokenVar, config.Token))
	cmd.Env = append(cmd.Env, "NO_BROWSER=1", "CI=1")
	cmd.Env = append(cmd.Env, "NODE_OPTIONS=--max-old-space-size=4096")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	// Write prompt to stdin asynchronously
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, prompt+"\n")
	}()

	return stdout, stderr, nil
}

// Helper to read stderr for debugging
func captureStderr(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	var sb strings.Builder
	for scanner.Scan() {
		sb.WriteString(scanner.Text() + "\n")
		if sb.Len() > 2000 { // Cap log size
			break
		}
	}
	return sb.String()
}
