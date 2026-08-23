package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/execution"
	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/queue"
)

type Pool struct {
	logger      *slog.Logger
	queue       queue.JobQueue
	store       jobs.JobStore
	execService *execution.Service
	numWorkers  int64

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewPool(logger *slog.Logger, q queue.JobQueue, s jobs.JobStore, e *execution.Service, workers int64) *Pool {
	return &Pool{
		logger:      logger,
		queue:       q,
		store:       s,
		execService: e,
		numWorkers:  workers,
	}
}

func (p *Pool) Start(ctx context.Context) {
	// Create a context specifically for the worker pool's lifetime
	poolCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	for i := int64(0); i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.workerLoop(poolCtx, i)
	}

	p.logger.Info("Worker pool started", "workers", p.numWorkers)
}

func (p *Pool) workerLoop(ctx context.Context, id int64) {
	defer p.wg.Done()
	log := p.logger.With("worker_id", id)
	log.Debug("Worker started")

	for {
		// Wait for a job to arrive or context to be cancelled
		jobID, err := p.queue.Dequeue(ctx)
		if err != nil {
			if err == queue.ErrQueueClosed || err == context.Canceled {
				log.Debug("Worker stopping cleanly")
				return
			}
			log.Error("Failed to dequeue job", "error", err)
			continue
		}

		p.processJobSafely(jobID, log)
	}
}

func (p *Pool) processJobSafely(jobID string, log *slog.Logger) {
	// Protect against panics in the execution pipeline so the worker doesn't die
	defer func() {
		if r := recover(); r != nil {
			log.Error("Worker recovered from panic", "job_id", jobID, "panic", r)
			
			// Attempt to mark the job as internally failed
			res := domain.ExecutionResult{
				Status:   domain.StatusRuntimeError,
				Stderr:   "Internal Platform Error: Worker Panic",
				ExitCode: -1,
			}
			_ = p.store.Complete(context.Background(), jobID, res)
		}
	}()

	log.Info("Processing job", "job_id", jobID)
	
	// Create a new context for this specific execution
	execCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // generous upper bound
	defer cancel()

	// 1. Mark Running
	if err := p.store.MarkRunning(execCtx, jobID); err != nil {
		log.Error("Failed to mark job as running", "job_id", jobID, "error", err)
		return
	}

	// 2. Fetch full job details
	job, err := p.store.Get(execCtx, jobID)
	if err != nil {
		log.Error("Failed to fetch job", "job_id", jobID, "error", err)
		return
	}

	// 3. Map to ExecutionRequest
	req := domain.ExecutionRequest{
		Language:   job.Language,
		SourceCode: job.SourceCode,
		Input:      job.Input,
	}

	// 4. Execute
	result, err := p.execService.Execute(execCtx, req)
	if err != nil {
		log.Error("Execution failed completely", "job_id", jobID, "error", err)
		result = domain.ExecutionResult{
			Status:   domain.StatusRuntimeError, // Or a dedicated InternalError status if we add one
			Stderr:   err.Error(),
			ExitCode: -1,
		}
	}

	// 5. Complete
	if err := p.store.Complete(execCtx, jobID, result); err != nil {
		log.Error("Failed to mark job as completed", "job_id", jobID, "error", err)
	}

	log.Info("Finished job", "job_id", jobID, "status", result.Status)
}

func (p *Pool) Stop(ctx context.Context) {
	p.logger.Info("Stopping worker pool...")
	
	// Close the queue so no more jobs can be enqueued
	p.queue.Close()
	
	// Signal all workers to stop (if they are waiting on a Dequeue with ctx)
	// Actually, closing the queue will cause Dequeue to return ErrQueueClosed,
	// but cancelling the context ensures any blocked operations wake up.
	if p.cancel != nil {
		p.cancel()
	}

	// Wait for active workers to finish, or until the passed context times out
	waitCh := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		p.logger.Info("All workers gracefully stopped")
	case <-ctx.Done():
		p.logger.Warn("Timeout waiting for workers to stop; forcefully exiting")
	}
}
