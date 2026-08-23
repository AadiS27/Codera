package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/platform/process"
	"github.com/codera/code-executor/internal/sandbox"
)

type Runtime struct {
	config *config.Config
}

func NewRuntime(cfg *config.Config) *Runtime {
	return &Runtime{config: cfg}
}

func (r *Runtime) StartContainer(ctx context.Context, workspace string) (string, error) {
	// Construct the docker run command to start a detached sandbox container
	args := []string{
		"run", "-d", "--rm",
		"--network", "none",
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--memory", r.config.ExecutionMemory,
		"--cpus", r.config.ExecutionCPUs,
		"--pids-limit", strconv.FormatInt(r.config.ExecutionPidsLimit, 10),
		"-v", fmt.Sprintf("%s:/workspace", workspace), // Mount workspace
		r.config.JavaSandboxImage,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to start sandbox container: %w, output: %s", err, string(out))
	}

	containerID := strings.TrimSpace(string(out))
	if len(containerID) == 0 {
		return "", fmt.Errorf("started sandbox container but got empty ID")
	}

	return containerID, nil
}

func (r *Runtime) Exec(ctx context.Context, containerID string, opts sandbox.ExecOptions) sandbox.ExecResult {
	// We use the same ProcessRunner from Phase 2 but inject "docker exec" instead!
	args := []string{
		"exec", "-i",
		"--user", "executor", // Enforce non-root user
		"-w", "/workspace", // Set working directory
		containerID,
		opts.Command,
	}
	args = append(args, opts.Args...)

	runOpts := process.RunOptions{
		Ctx:         ctx,
		Command:     "docker",
		Args:        args,
		Dir:         "", // Not needed since docker exec runs globally
		Stdin:       opts.Stdin,
		StdoutLimit: opts.StdoutLimit,
		StderrLimit: opts.StderrLimit,
		Timeout:     opts.Timeout,
	}

	res := process.RunProcess(runOpts)

	return sandbox.ExecResult{
		Stdout:      res.Stdout,
		Stderr:      res.Stderr,
		ExitCode:    res.ExitCode,
		Timeout:     res.Timeout,
		OutputLimit: res.OutputLimit,
		Error:       res.Error,
	}
}

func (r *Runtime) DestroyContainer(containerID string) error {
	if containerID == "" {
		return nil
	}

	// Use a background context with a generous timeout to ensure cleanup runs
	// even if the incoming request context is already cancelled (e.g. by timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerID)
	return cmd.Run()
}
