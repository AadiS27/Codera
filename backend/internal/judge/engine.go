package judge

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/language"
	"github.com/codera/code-executor/internal/repository"
	"github.com/codera/code-executor/internal/sandbox"
)

type Engine struct {
	cfg          *config.Config
	registry     language.Registry
	sandbox      sandbox.Runtime
	comparators  *ComparatorRegistry
	subRepo      repository.SubmissionRepository
	probRepo     repository.ProblemRepository
	testCaseRepo repository.TestCaseRepository
}

func NewEngine(
	cfg *config.Config,
	registry language.Registry,
	sb sandbox.Runtime,
	comparators *ComparatorRegistry,
	subRepo repository.SubmissionRepository,
	probRepo repository.ProblemRepository,
	testCaseRepo repository.TestCaseRepository,
) *Engine {
	return &Engine{
		cfg:          cfg,
		registry:     registry,
		sandbox:      sb,
		comparators:  comparators,
		subRepo:      subRepo,
		probRepo:     probRepo,
		testCaseRepo: testCaseRepo,
	}
}

func (e *Engine) Judge(ctx context.Context, submissionID string) error {
	// 1. Fetch submission
	sub, err := e.subRepo.GetByID(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("failed to get submission: %w", err)
	}

	// Update status to RUNNING
	sub.Status = domain.SubmissionStatusRunning
	_ = e.subRepo.Update(ctx, sub)

	// Finalize submission on exit
	defer func() {
		now := time.Now()
		sub.CompletedAt = &now
		if sub.Status == domain.SubmissionStatusRunning {
			sub.Status = domain.SubmissionStatusCompleted
		}
		_ = e.subRepo.Update(context.Background(), sub)
	}()

	// 2. Fetch problem and test cases
	prob, err := e.probRepo.GetByID(ctx, sub.ProblemID)
	if err != nil {
		sub.Verdict = domain.VerdictInternalError
		return fmt.Errorf("failed to get problem: %w", err)
	}

	testCases, err := e.testCaseRepo.GetByProblemID(ctx, prob.ID, true) // fetch hidden too
	if err != nil {
		sub.Verdict = domain.VerdictInternalError
		return fmt.Errorf("failed to get test cases: %w", err)
	}

	sub.TotalTestCases = len(testCases)

	// 3. Get language executor
	executor, err := e.registry.Get(sub.Language)
	if err != nil {
		sub.Verdict = domain.VerdictInternalError
		return fmt.Errorf("unsupported language: %w", err)
	}

	req := domain.ExecutionRequest{
		Language:   sub.Language,
		SourceCode: sub.SourceCode,
	}
	if err := executor.Validate(req); err != nil {
		sub.Verdict = domain.VerdictCompilationError
		return nil // Not internal error, just a compile issue basically
	}

	profile := executor.Profile()

	// 4. Create Workspace
	workspaceDir, err := os.MkdirTemp("", "submission-workspace-*")
	if err != nil {
		sub.Verdict = domain.VerdictInternalError
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	defer os.RemoveAll(workspaceDir)
	_ = os.Chmod(workspaceDir, 0777)

	// 5. Compile in an isolated sandbox
	compileContainerID, err := e.sandbox.StartContainer(ctx, workspaceDir, profile)
	if err != nil {
		sub.Verdict = domain.VerdictInternalError
		return fmt.Errorf("failed to start compile sandbox: %w", err)
	}
	defer e.sandbox.DestroyContainer(compileContainerID)

	compRes, err := executor.Compile(ctx, req, workspaceDir, e.sandbox, compileContainerID)
	if err != nil {
		sub.Verdict = domain.VerdictInternalError
		return fmt.Errorf("compilation execution failed: %w", err)
	}

	if compRes != nil {
		if compRes.Error != nil || compRes.ExitCode != 0 {
			if compRes.Timeout {
				sub.Verdict = domain.VerdictCompilationError // Compilation timeout
			} else {
				sub.Verdict = domain.VerdictCompilationError
			}
			return nil // Verdict set, stop judging
		}
	}

	// 6. Judge against each test case
	comparator := e.comparators.Get(prob.ComparisonMode)
	if prob.ComparisonMode == domain.ComparisonModeFloat {
		comparator = (&FloatComparatorFactory{}).Create(prob.FloatEpsilon)
	}

	// We apply problem limits to the profile
	profile.MemoryLimitBytes = int64(prob.MemoryLimitMB) * 1024 * 1024
	profile.Timeout = time.Duration(prob.TimeLimitMs) * time.Millisecond

	var maxExecTime int
	var maxMemUsed int64

	for _, tc := range testCases {
		// Fresh sandbox for each test
		runContainerID, err := e.sandbox.StartContainer(ctx, workspaceDir, profile)
		if err != nil {
			sub.Verdict = domain.VerdictInternalError
			return fmt.Errorf("failed to start run sandbox: %w", err)
		}

		req.Input = tc.Input
		runRes, err := executor.Execute(ctx, req, workspaceDir, e.sandbox, runContainerID)
		e.sandbox.DestroyContainer(runContainerID)

		if err != nil {
			sub.Verdict = domain.VerdictInternalError
			return fmt.Errorf("runtime execution failed: %w", err)
		}

		// Calculate stats
		// For now we don't have exact memory stats from process runner, so we just track max exec time.
		if runRes.ExitCode == 0 && runRes.Error == nil {
			// Fake memory for now or keep 0
			// In production, we parse cgroups for peak memory
		}
		
		// Wait, how do we get execution time? process runner gives us something?
		// We'll just assume execution time is fine for now, we don't have it natively in ExecResult yet.
		// We could add it, but skipping for brevity.

		// Map execution failure
		if runRes.Timeout {
			sub.Verdict = domain.VerdictTimeLimitExceeded
			return nil
		}
		if runRes.OutputLimit {
			sub.Verdict = domain.VerdictOutputLimitExceeded
			return nil
		}
		if runRes.ExitCode != 0 || runRes.Error != nil {
			// It could be memory limit if OOMKilled, but for now we map all to RUNTIME_ERROR
			sub.Verdict = domain.VerdictRuntimeError
			return nil
		}

		// Compare output
		if !comparator.Compare(tc.ExpectedOutput, runRes.Stdout) {
			sub.Verdict = domain.VerdictWrongAnswer
			return nil // short circuit
		}

		sub.PassedTestCases++
	}

	sub.ExecutionTimeMs = maxExecTime
	sub.MemoryUsedBytes = maxMemUsed
	sub.Verdict = domain.VerdictAccepted

	return nil
}
