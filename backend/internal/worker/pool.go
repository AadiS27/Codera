package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/execution"
	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/judge"
	"github.com/codera/code-executor/internal/queue"
)

type Pool struct {
	logger      *slog.Logger
	queue       queue.JobQueue
	store       *jobs.PostgresJobStore
	engine      *judge.Engine
	execService *execution.Service
	numWorkers  int64
	instanceID  string
	heartbeat   *Heartbeat

	wg           sync.WaitGroup
	cancel       context.CancelFunc
	shutdownTimeout time.Duration
	status       WorkerStatus
}

func NewPool(logger *slog.Logger, q queue.JobQueue, s *jobs.PostgresJobStore, eng *judge.Engine, e *execution.Service, workers int64, instanceID string, hbInterval, shutdownTimeout time.Duration) *Pool {
	return &Pool{
		logger:          logger,
		queue:           q,
		store:           s,
		engine:          eng,
		execService:     e,
		numWorkers:      workers,
		instanceID:      instanceID,
		heartbeat:       NewHeartbeat(s.Pool(), logger, instanceID, hbInterval),
		shutdownTimeout: shutdownTimeout,
		status:          StatusStarting,
	}
}

func (p *Pool) Start(ctx context.Context) {
	poolCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.status = StatusActive

	// Start Heartbeat
	if err := p.heartbeat.UpdateStatus(poolCtx, StatusActive); err != nil {
		p.logger.Error("Failed to update initial heartbeat status", "error", err)
	}
	go p.heartbeat.Run(poolCtx)

	for i := int64(0); i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.workerLoop(poolCtx, i)
	}

	p.logger.Info("Worker pool started", "workers", p.numWorkers, "instance_id", p.instanceID)
}

func (p *Pool) workerLoop(ctx context.Context, id int64) {
	defer p.wg.Done()
	
	workerName := fmt.Sprintf("%s-worker-%d", p.instanceID, id)
	log := p.logger.With("worker_id", workerName)
	log.Debug("Worker started")

	for {
		if p.status == StatusDraining || p.status == StatusStopped {
			log.Debug("Worker not accepting new jobs due to shutdown")
			return
		}

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
			if err := p.store.FailJob(context.Background(), msg.JobID, workerName, "Worker crashed during execution", 0); err != nil {
				p.logger.Error("Failed to record FailJob in DB during recovery", "job_id", msg.JobID, "error", err)
			}
		}
	}()

	log.Info("Processing job", "job_id", msg.JobID, "job_type", msg.JobType, "msg_id", msg.ID)
	
	if msg.JobType == "submission" {
		p.processSubmission(ctx, msg, workerName, log)
	} else {
		p.processRun(ctx, msg, workerName, log)
	}
}

func (p *Pool) processSubmission(ctx context.Context, msg queue.QueueMessage, workerName string, log *slog.Logger) {
	// For submissions, the JudgeEngine takes care of everything (status updates, completion, etc)
	// We just need to give it context.
	err := p.engine.Judge(ctx, msg.JobID)
	if err != nil {
		log.Error("JudgeEngine failed", "submission_id", msg.JobID, "error", err)
		// We could implement retry logic, but JudgeEngine saves verdicts.
	}
	
	if err := p.queue.Ack(context.Background(), msg.ID); err != nil {
		log.Error("Failed to ACK redis message for submission", "msg_id", msg.ID, "error", err)
	}
}

func (p *Pool) processRun(ctx context.Context, msg queue.QueueMessage, workerName string, log *slog.Logger) {
	leaseDuration := 30 * time.Second

	// 1. Claim Job
	err := p.store.Claim(ctx, msg.JobID, workerName, leaseDuration)
	if err != nil {
		if err == jobs.ErrInvalidState {
			job, getErr := p.store.Get(ctx, msg.JobID)
			if getErr == nil && (job.Status == jobs.StatusCompleted || job.Status == jobs.StatusDeadLettered) {
				log.Debug("Received message for completed/dead job, ACKing", "job_id", msg.JobID)
				_ = p.queue.Ack(context.Background(), msg.ID)
			} else {
				log.Debug("Failed to claim job (already running or invalid state), ignoring message", "job_id", msg.JobID)
			}
			return
		}
		log.Error("Failed to claim job", "job_id", msg.JobID, "error", err)
		return
	}

	// 2. Mark Running
	if err := p.store.MarkRunning(ctx, msg.JobID, workerName); err != nil {
		log.Error("Failed to mark running", "job_id", msg.JobID, "error", err)
		return
	}

	// 2. Fetch full job details
	job, err := p.store.Get(ctx, msg.JobID)
	if err != nil {
		log.Error("Failed to fetch job", "job_id", msg.JobID, "error", err)
		return
	}

	// Publish RUNNING update
	if payload, err := json.Marshal(job); err == nil {
		_ = p.queue.PublishJobUpdate(context.Background(), msg.JobID, payload)
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

	// 3. Map to ExecutionRequest and execute for each input
	var results []domain.ExecutionResult
	for _, input := range job.Inputs {
		req := domain.ExecutionRequest{
			Language:   job.Language,
			SourceCode: job.SourceCode,
			Input:      input,
		}

		result, err := p.execService.Execute(execCtx, req)
		if err != nil {
			if execCtx.Err() != nil {
				log.Warn("Execution cancelled locally (lease lost or shutdown)", "job_id", msg.JobID)
				return
			}

			log.Error("Platform Execution failed", "job_id", msg.JobID, "error", err)
			// Platform failure (Docker missing, internal error) -> FailJob for retry
			backoff := 2 * time.Second // Simple backoff for now, can be improved in production
			if failErr := p.store.FailJob(context.Background(), msg.JobID, workerName, err.Error(), backoff); failErr != nil {
				log.Error("Failed to record FailJob in DB", "job_id", msg.JobID, "error", failErr)
			}
			return // Let the Redis message un-ACK so pending recovery or DLQ handles it
		}
		
		results = append(results, result)
	}

	// For user failures (StatusCompilationError, StatusRuntimeError, etc), we Complete normally
	// 5. Complete in DB
	if err := p.store.Complete(context.Background(), msg.JobID, workerName, results); err != nil {
		log.Error("Failed to mark job as completed in DB", "job_id", msg.JobID, "error", err)
		return
	}

	// Fetch updated job to publish
	if updatedJob, err := p.store.Get(context.Background(), msg.JobID); err == nil {
		if payload, err := json.Marshal(updatedJob); err == nil {
			_ = p.queue.PublishJobUpdate(context.Background(), msg.JobID, payload)
		}
	}

	// 6. ACK Redis message
	if err := p.queue.Ack(context.Background(), msg.ID); err != nil {
		log.Error("Failed to ACK redis message", "msg_id", msg.ID, "error", err)
	}

	log.Info("Finished job", "job_id", msg.JobID, "num_results", len(results))
}

func (p *Pool) Stop(ctx context.Context) {
	p.logger.Info("Stopping worker pool, entering DRAINING state...")
	
	p.status = StatusDraining
	_ = p.heartbeat.UpdateStatus(context.Background(), StatusDraining)
	
	// Close the queue so no more jobs can be enqueued or consumed
	p.queue.Close()

	waitCh := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(waitCh)
	}()

	// Wait up to ShutdownTimeout for active jobs to finish
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), p.shutdownTimeout)
	defer cancelShutdown()

	select {
	case <-waitCh:
		p.logger.Info("All workers gracefully drained")
	case <-shutdownCtx.Done():
		p.logger.Warn("Timeout waiting for workers to drain; cancelling remaining contexts")
		if p.cancel != nil {
			p.cancel()
		}
		// Wait one last time briefly after cancel
		time.Sleep(1 * time.Second)
	}

	p.status = StatusStopped
	_ = p.heartbeat.UpdateStatus(context.Background(), StatusStopped)
}
