package worker

import (
	"context"
	"fmt"
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
	store       *jobs.PostgresJobStore
	execService *execution.Service
	numWorkers  int64
	instanceID  string

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewPool(logger *slog.Logger, q queue.JobQueue, s *jobs.PostgresJobStore, e *execution.Service, workers int64, instanceID string) *Pool {
	return &Pool{
		logger:      logger,
		queue:       q,
		store:       s,
		execService: e,
		numWorkers:  workers,
		instanceID:  instanceID,
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
	
	workerName := fmt.Sprintf("%s-worker-%d", p.instanceID, id)
	log := p.logger.With("worker_id", workerName)
	log.Debug("Worker started")

	for {
		// Wait for a job to arrive or context to be cancelled
		msg, err := p.queue.Consume(ctx, workerName)
		if err != nil {
			if err == queue.ErrQueueClosed || err == context.Canceled {
				log.Debug("Worker stopping cleanly")
				return
			}
			log.Error("Failed to dequeue job", "error", err)
			time.Sleep(1 * time.Second) // backoff
			continue
		}

		p.processJobSafely(ctx, msg, workerName, log)
	}
}

func (p *Pool) processJobSafely(ctx context.Context, msg queue.QueueMessage, workerName string, log *slog.Logger) {
	// Protect against panics in the execution pipeline so the worker doesn't die
	defer func() {
		if r := recover(); r != nil {
			log.Error("Worker recovered from panic", "job_id", msg.JobID, "panic", r)
			
			// Attempt to mark the job as internally failed
			res := domain.ExecutionResult{
				Status:   domain.StatusRuntimeError,
				Stderr:   "Internal Platform Error: Worker Panic",
				ExitCode: -1,
			}
			_ = p.store.Complete(context.Background(), msg.JobID, workerName, res)
		}
	}()

	log.Info("Processing job", "job_id", msg.JobID, "msg_id", msg.ID)
	
	leaseDuration := 30 * time.Second

	// 1. Mark Running (Atomically Claim via PostgreSQL)
	err := p.store.MarkRunning(ctx, msg.JobID, workerName, leaseDuration)
	if err != nil {
		if err == jobs.ErrInvalidState {
			// This could mean the job is already RUNNING (another worker got it first)
			// OR the job is already COMPLETED (stale duplicate message).
			// Let's check the database state.
			job, getErr := p.store.Get(ctx, msg.JobID)
			if getErr == nil && job.Status == jobs.StatusCompleted {
				// It's completed. Safe to ACK this duplicate message.
				log.Debug("Received duplicate message for completed job, ACKing", "job_id", msg.JobID)
				_ = p.queue.Ack(context.Background(), msg.ID)
			} else {
				log.Debug("Failed to claim job (already running or invalid state), ignoring message", "job_id", msg.JobID)
			}
			return
		}
		log.Error("Failed to mark job as running", "job_id", msg.JobID, "error", err)
		return
	}

	// 2. Fetch full job details
	job, err := p.store.Get(ctx, msg.JobID)
	if err != nil {
		log.Error("Failed to fetch job", "job_id", msg.JobID, "error", err)
		return
	}

	// Create an execution context that is cancelled if worker stops or lease renewal fails
	execCtx, cancelExec := context.WithCancel(ctx)
	defer cancelExec()

	// Start lease renewal goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Second) // Renew every 10s
		defer ticker.Stop()
		for {
			select {
			case <-execCtx.Done():
				return
			case <-ticker.C:
				if err := p.store.RenewLease(context.Background(), msg.JobID, workerName, leaseDuration); err != nil {
					log.Error("Failed to renew lease! Cancelling local execution", "job_id", msg.JobID, "error", err)
					cancelExec()
					return
				}
			}
		}
	}()

	// 3. Map to ExecutionRequest
	req := domain.ExecutionRequest{
		Language:   job.Language,
		SourceCode: job.SourceCode,
		Input:      job.Input,
	}

	// 4. Execute
	result, err := p.execService.Execute(execCtx, req)
	if err != nil {
		if execCtx.Err() != nil {
			// Execution was cancelled locally (e.g., shutdown or lease lost).
			log.Warn("Execution cancelled locally (lease lost or shutdown)", "job_id", msg.JobID)
			// DO NOT ACK. Let the lease expire naturally so another worker can recover it.
			return
		}

		log.Error("Execution failed completely", "job_id", msg.JobID, "error", err)
		result = domain.ExecutionResult{
			Status:   domain.StatusRuntimeError,
			Stderr:   err.Error(),
			ExitCode: -1,
		}
	}

	// 5. Complete in DB
	if err := p.store.Complete(context.Background(), msg.JobID, workerName, result); err != nil {
		log.Error("Failed to mark job as completed in DB", "job_id", msg.JobID, "error", err)
		return // Do not ACK if DB completion failed
	}

	// 6. ACK Redis message
	if err := p.queue.Ack(context.Background(), msg.ID); err != nil {
		log.Error("Failed to ACK redis message", "msg_id", msg.ID, "error", err)
	}

	log.Info("Finished job", "job_id", msg.JobID, "status", result.Status)
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
