package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/platform/process"
	"github.com/codera/code-executor/internal/sandbox"
)

type Runtime struct {
	config      *config.Config
	seccompPath string
}

func NewRuntime(cfg *config.Config) *Runtime {
	// Write a basic restricted seccomp profile to a temp file so Docker can use it
	seccompPath := createSeccompProfile()
	return &Runtime{config: cfg, seccompPath: seccompPath}
}

func createSeccompProfile() string {
	// In production, this would be loaded from a file or embedded.
	// We use a highly restrictive base for code execution.
	// (Excluding dangerous syscalls like mount, unshare, ptrace, bpf, etc)
	content := `{
		"defaultAction": "SCMP_ACT_ALLOW",
		"architectures": ["SCMP_ARCH_X86_64", "SCMP_ARCH_AARCH64"],
		"syscalls": [
			{
				"names": [
					"mount", "umount2", "ptrace", "unshare", "setns", "bpf",
					"kexec_load", "kexec_file_load", "init_module", "finit_module",
					"delete_module", "create_module", "swapon", "swapoff", "reboot",
					"sethostname", "setdomainname", "iopl", "ioperm", "nfsservctl",
					"syslog", "vhangup", "pivot_root", "acct", "quotactl", "perf_event_open",
					"process_vm_readv", "process_vm_writev"
				],
				"action": "SCMP_ACT_ERRNO"
			}
		]
	}`
	
	tmpfile, err := os.CreateTemp("", "seccomp-*.json")
	if err != nil {
		panic("failed to create temp seccomp file: " + err.Error())
	}
	_, _ = tmpfile.WriteString(content)
	tmpfile.Close()
	return tmpfile.Name()
}

func (r *Runtime) StartContainer(ctx context.Context, workspace string, profile sandbox.Profile) (string, error) {
	// Construct the docker run command to start a detached sandbox container
	args := []string{
		"run", "-d", "--rm",
		"--network", "none",
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--security-opt", "seccomp=" + r.seccompPath,
		"--memory", strconv.FormatInt(profile.MemoryLimitBytes, 10),
		"--cpus", strconv.FormatFloat(profile.CPULimit, 'f', -1, 64),
		"--pids-limit", strconv.FormatInt(profile.PidsLimit, 10),
		"--tmpfs", fmt.Sprintf("/tmp:rw,size=%d,mode=1777", profile.TmpfsSizeBytes),
		"-v", fmt.Sprintf("%s:/workspace", workspace), // Mount workspace
		profile.Image,
		"sleep", "infinity",
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
		"--user", "1000:1000", // Enforce non-root user (numeric to avoid /etc/passwd dependence)
		"-w", "/workspace", // Set working directory to workspace
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
