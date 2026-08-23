package docker

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/sandbox"
)

// ensureDockerRunning skips the test if Docker is not available
func ensureDockerRunning(t *testing.T) {
	// Our runtime commands require docker on the host
}

// getTestRuntime creates a Runtime with default config
func getTestRuntime(t *testing.T) *Runtime {
	cfg := config.DefaultConfig()
	return NewRuntime(cfg)
}

func TestSandboxNonRoot(t *testing.T) {
	ensureDockerRunning(t)
	rt := getTestRuntime(t)
	ctx := context.Background()

	// Use python image as the sandbox
	profile := sandbox.ProfileSmall
	profile.Image = "python:3.11-alpine"

	containerID, err := rt.StartContainer(ctx, "/tmp", profile)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer rt.DestroyContainer(containerID)

	// Run 'id -u'
	opts := sandbox.ExecOptions{
		Command:     "id",
		Args:        []string{"-u"},
		StdoutLimit: 1024,
		StderrLimit: 1024,
		Timeout:     2 * time.Second,
	}
	res := rt.Exec(ctx, containerID, opts)

	if res.Error != nil {
		t.Fatalf("Exec failed: %v", res.Error)
	}
	
	uid := strings.TrimSpace(res.Stdout)
	if uid == "0" {
		t.Fatalf("Sandbox is running as root (UID 0)!")
	}
}

func TestSandboxNoNetwork(t *testing.T) {
	ensureDockerRunning(t)
	rt := getTestRuntime(t)
	ctx := context.Background()

	profile := sandbox.ProfileSmall
	profile.Image = "python:3.11-alpine"

	containerID, err := rt.StartContainer(ctx, "/tmp", profile)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer rt.DestroyContainer(containerID)

	// Try to ping google
	opts := sandbox.ExecOptions{
		Command:     "ping",
		Args:        []string{"-c", "1", "8.8.8.8"},
		StdoutLimit: 1024,
		StderrLimit: 1024,
		Timeout:     2 * time.Second,
	}
	res := rt.Exec(ctx, containerID, opts)

	if res.ExitCode == 0 {
		t.Fatalf("Ping succeeded! Network is NOT isolated.")
	}
}

func TestSandboxReadOnlyRoot(t *testing.T) {
	ensureDockerRunning(t)
	rt := getTestRuntime(t)
	ctx := context.Background()

	profile := sandbox.ProfileSmall
	profile.Image = "python:3.11-alpine"

	containerID, err := rt.StartContainer(ctx, "/tmp", profile)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer rt.DestroyContainer(containerID)

	// Attempt to touch a file in root
	opts := sandbox.ExecOptions{
		Command:     "touch",
		Args:        []string{"/etc/malicious"},
		StdoutLimit: 1024,
		StderrLimit: 1024,
		Timeout:     2 * time.Second,
	}
	res := rt.Exec(ctx, containerID, opts)

	if res.ExitCode == 0 {
		t.Fatalf("Successfully wrote to read-only root filesystem!")
	}
	if !strings.Contains(res.Stderr, "Read-only file system") {
		t.Logf("Expected Read-only file system error, got: %s", res.Stderr)
	}
}

func TestSandboxMemoryBomb(t *testing.T) {
	ensureDockerRunning(t)
	rt := getTestRuntime(t)
	ctx := context.Background()

	profile := sandbox.ProfileSmall
	profile.Image = "python:3.11-alpine"
	profile.MemoryLimitBytes = 32 * 1024 * 1024 // 32MB

	containerID, err := rt.StartContainer(ctx, "/tmp", profile)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer rt.DestroyContainer(containerID)

	// Python script that allocates memory infinitely
	script := "data = []\nwhile True:\n    data.append(b'x' * 1024 * 1024)\n"
	
	opts := sandbox.ExecOptions{
		Command:     "python",
		Args:        []string{"-c", script},
		StdoutLimit: 1024,
		StderrLimit: 1024,
		Timeout:     5 * time.Second,
	}
	res := rt.Exec(ctx, containerID, opts)

	if res.ExitCode == 0 {
		t.Fatalf("Memory bomb succeeded without being killed!")
	}
	
	// Docker usually exits with 137 (OOM Killed)
	if res.ExitCode != 137 && res.ExitCode != 139 && !strings.Contains(res.Stderr, "MemoryError") {
		t.Logf("Expected OOM kill (137) or MemoryError, got ExitCode %d", res.ExitCode)
	}
}

func TestSandboxDiskFill(t *testing.T) {
	ensureDockerRunning(t)
	rt := getTestRuntime(t)
	ctx := context.Background()

	profile := sandbox.ProfileSmall
	profile.Image = "python:3.11-alpine"
	profile.TmpfsSizeBytes = 10 * 1024 * 1024 // 10MB tmpfs

	containerID, err := rt.StartContainer(ctx, "/tmp", profile)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer rt.DestroyContainer(containerID)

	// Python script that fills disk at /tmp
	script := `
with open('/tmp/bigfile', 'wb') as f:
    while True:
        f.write(b'x' * 1024 * 1024)
`
	
	opts := sandbox.ExecOptions{
		Command:     "python",
		Args:        []string{"-c", script},
		StdoutLimit: 1024,
		StderrLimit: 1024,
		Timeout:     3 * time.Second,
	}
	res := rt.Exec(ctx, containerID, opts)

	if res.ExitCode == 0 {
		t.Fatalf("Disk fill succeeded without OS limit!")
	}
	
	if !strings.Contains(res.Stderr, "No space left on device") {
		t.Fatalf("Expected 'No space left on device', got: %s", res.Stderr)
	}
}

func TestSandboxForkBomb(t *testing.T) {
	ensureDockerRunning(t)
	rt := getTestRuntime(t)
	ctx := context.Background()

	profile := sandbox.ProfileSmall
	profile.Image = "python:3.11-alpine"
	profile.PidsLimit = 32

	containerID, err := rt.StartContainer(ctx, "/tmp", profile)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer rt.DestroyContainer(containerID)

	// Python script that attempts to fork infinitely
	script := `
import os
import time
try:
    while True:
        os.fork()
except OSError as e:
    print(f"Fork failed: {e}")
    time.sleep(1) # stay alive so we can capture output
`
	
	opts := sandbox.ExecOptions{
		Command:     "python",
		Args:        []string{"-c", script},
		StdoutLimit: 1024,
		StderrLimit: 1024,
		Timeout:     3 * time.Second, // Let it hit the fork limit, then sleep and get killed or exit
	}
	res := rt.Exec(ctx, containerID, opts)

	if !strings.Contains(res.Stdout, "Fork failed") {
		t.Fatalf("Expected fork to fail due to PID limit, got exit code: %d, output: %s %s", res.ExitCode, res.Stdout, res.Stderr)
	}
}
