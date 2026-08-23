package classifier

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/sandbox"
)

// Classify takes the raw sandbox result and maps it to the domain ExecutionStatus
func Classify(res sandbox.ExecResult) domain.ExecutionResult {
	if res.Error != nil {
		// If it's an ExitError, it means the process ran but exited non-zero.
		// We shouldn't treat this as an internal platform error.
		var exitErr *exec.ExitError
		if !errors.As(res.Error, &exitErr) {
			return domain.ExecutionResult{
				Status:   domain.StatusInternalError,
				Stderr:   "Internal Platform Error: " + res.Error.Error(),
				ExitCode: -1,
			}
		}
	}

	if res.Timeout {
		return domain.ExecutionResult{
			Status:   domain.StatusTimeLimitExceeded,
			Stdout:   res.Stdout,
			Stderr:   res.Stderr,
			ExitCode: res.ExitCode,
		}
	}

	if res.OutputLimit {
		return domain.ExecutionResult{
			Status:   domain.StatusOutputLimitExceeded,
			Stdout:   res.Stdout,
			Stderr:   res.Stderr,
			ExitCode: res.ExitCode,
		}
	}

	// Exit code 137 typically indicates the process was killed by SIGKILL.
	// In the context of Docker with memory limits, it's often the OOM killer.
	if res.ExitCode == 137 {
		// We could check if it was timeout, but that's already handled above.
		// If it's 137 and not a timeout, it's highly likely an OOM kill.
		return domain.ExecutionResult{
			Status:   domain.StatusMemoryLimitExceeded,
			Stdout:   res.Stdout,
			Stderr:   "Memory Limit Exceeded", // Custom message or include raw stderr
			ExitCode: 137,
		}
	}

	if res.ExitCode != 0 {
		// Depending on whether it's compile time or run time, it could be COMPILATION_ERROR.
		// For now, assume it's RUNTIME_ERROR since compilation steps aren't separated in a multi-stage yet.
		// If you separate compilation, you would classify it there.
		
		// If stderr contains specific memory/alloc panics, we can also classify as MemoryLimitExceeded
		if strings.Contains(res.Stderr, "java.lang.OutOfMemoryError") || strings.Contains(res.Stderr, "MemoryError") || strings.Contains(res.Stderr, "out of memory") {
			return domain.ExecutionResult{
				Status:   domain.StatusMemoryLimitExceeded,
				Stdout:   res.Stdout,
				Stderr:   res.Stderr,
				ExitCode: res.ExitCode,
			}
		}

		return domain.ExecutionResult{
			Status:   domain.StatusRuntimeError,
			Stdout:   res.Stdout,
			Stderr:   res.Stderr,
			ExitCode: res.ExitCode,
		}
	}

	return domain.ExecutionResult{
		Status:   domain.StatusSuccess,
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
		ExitCode: 0,
	}
}
