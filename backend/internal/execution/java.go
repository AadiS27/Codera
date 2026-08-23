package execution

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/sandbox"
	"github.com/codera/code-executor/internal/sandbox/classifier"
)

type JavaExecutor struct {
	config  *config.Config
	sandbox sandbox.Runtime
}

func NewJavaExecutor(cfg *config.Config, sb sandbox.Runtime) *JavaExecutor {
	return &JavaExecutor{config: cfg, sandbox: sb}
}


func (j *JavaExecutor) Execute(ctx context.Context, req domain.ExecutionRequest) (domain.ExecutionResult, error) {
	// Step 1: Create a unique temporary directory on host
	workspace, err := os.MkdirTemp("", "code-executor-java-*")
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workspace) // Step 7: Cleanup workspace always

	// Step 2: Write source code
	sourcePath := filepath.Join(workspace, "Main.java")
	if err := os.WriteFile(sourcePath, []byte(req.SourceCode), 0777); err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to write source file: %w", err)
	}
	// Give full permissions so Docker non-root user can read/write to this mounted dir
	_ = os.Chmod(workspace, 0777)
	_ = os.Chmod(sourcePath, 0666)

	// Step 3: Get Profile and Start Sandbox Container
	profile, err := sandbox.GetProfileForLanguage("java")
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to get sandbox profile: %w", err)
	}

	containerID, err := j.sandbox.StartContainer(ctx, workspace, profile)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to start sandbox: %w", err)
	}
	defer j.sandbox.DestroyContainer(containerID)

	// Step 4: Compile
	compileOpts := sandbox.ExecOptions{
		Command:     "javac",
		Args:        []string{"-source", "1.8", "-target", "1.8", "Main.java"},
		StdoutLimit: profile.MaxOutputBytes,
		StderrLimit: profile.MaxOutputBytes,
		Timeout:     j.config.CompileTimeout, // Compile timeout is different from run timeout
	}

	compileRes := j.sandbox.Exec(ctx, containerID, compileOpts)
	
	// We classify the compile result. Since it's compile phase, we map RUNTIME_ERROR to COMPILATION_ERROR.
	compileDomainRes := classifier.Classify(compileRes)
	if compileDomainRes.Status != domain.StatusSuccess {
		if compileDomainRes.Status == domain.StatusRuntimeError {
			compileDomainRes.Status = domain.StatusCompilationError
		} else if compileDomainRes.Status == domain.StatusTimeLimitExceeded {
			compileDomainRes.Status = domain.StatusCompilationTimeout
		}
		return compileDomainRes, nil
	}

	// Step 5: Run
	var stdinReader io.Reader
	if req.Input != "" {
		stdinReader = strings.NewReader(req.Input)
	}

	runOpts := sandbox.ExecOptions{
		Command:     "java",
		Args:        []string{"Main"},
		Stdin:       stdinReader,
		StdoutLimit: profile.MaxOutputBytes,
		StderrLimit: profile.MaxOutputBytes,
		Timeout:     profile.Timeout,
	}

	runRes := j.sandbox.Exec(ctx, containerID, runOpts)
	runDomainRes := classifier.Classify(runRes)

	return runDomainRes, nil
}
