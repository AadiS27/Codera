package recovery

import (
	"context"
	"log/slog"
	"time"

	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool             *pgxpool.Pool
	logger           *slog.Logger
	queue            queue.JobQueue
	recoveryInterval time.Duration
	heartbeatTimeout time.Duration
}

func NewService(pool *pgxpool.Pool, logger *slog.Logger, queue queue.JobQueue, interval, hbTimeout time.Duration) *Service {
	return &Service{
		pool:             pool,
		logger:           logger,
		queue:            queue,
		recoveryInterval: interval,
		heartbeatTimeout: hbTimeout,
	}
}

func (s *Service) Start(ctx context.Context) {
	ticker := time.NewTicker(s.recoveryInterval)
	defer ticker.Stop()

	s.logger.Info("Recovery service started", "interval", s.recoveryInterval)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Recovery service stopped")
			return
		case <-ticker.C:
			s.RecoverAbandonedJobs(ctx)
			s.DispatchDelayedJobs(ctx)
			s.MarkStaleWorkers(ctx)
		}
	}
}

func (s *Service) RecoverAbandonedJobs(ctx context.Context) {
	// Atomically reclaim jobs with expired leases
	query := `
		UPDATE executions
		SET 
			job_status = CASE 
				WHEN attempt_count < max_attempts THEN $1::text 
				ELSE $2::text 
			END,
			worker_id = NULL,
			lease_expires_at = NULL,
			last_error = 'worker lease expired',
			next_retry_at = CASE
				WHEN attempt_count < max_attempts THEN NOW()
				ELSE NULL
			END,
			dead_lettered_at = CASE
				WHEN attempt_count >= max_attempts THEN NOW()
				ELSE NULL
			END
		WHERE job_status IN ($3, $4) AND lease_expires_at < NOW()
	`

	cmd, err := s.pool.Exec(ctx, query, 
		string(jobs.StatusQueued), 
		string(jobs.StatusDeadLettered), 
		string(jobs.StatusClaimed), 
		string(jobs.StatusRunning),
	)

	if err != nil {
		s.logger.Error("Failed to recover abandoned jobs", "error", err)
		return
	}

	if recovered := cmd.RowsAffected(); recovered > 0 {
		s.logger.Warn("Recovered abandoned jobs", "count", recovered)
	}
}

func (s *Service) DispatchDelayedJobs(ctx context.Context) {
	// Find QUEUED jobs ready for retry (next_retry_at <= NOW)
	// and that haven't been dispatched recently (to prevent storm)
	query := `
		SELECT id FROM executions
		WHERE job_status = $1
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		  AND (last_dispatched_at IS NULL OR last_dispatched_at < NOW() - INTERVAL '5 seconds')
		LIMIT 100
	`

	rows, err := s.pool.Query(ctx, query, string(jobs.StatusQueued))
	if err != nil {
		s.logger.Error("Failed to query delayed jobs for dispatch", "error", err)
		return
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			s.logger.Error("Failed to scan delayed job ID", "error", err)
			continue
		}
		ids = append(ids, id)
	}
	rows.Close() // Explicit close before starting dispatch

	for _, id := range ids {
		// Enqueue to Redis
		if err := s.queue.Enqueue(ctx, id); err != nil {
			s.logger.Error("Failed to enqueue delayed job", "job_id", id, "error", err)
			continue
		}

		// Update dispatch marker
		updateQuery := `UPDATE executions SET last_dispatched_at = NOW() WHERE id = $1`
		_, err := s.pool.Exec(ctx, updateQuery, id)
		if err != nil {
			s.logger.Error("Failed to update last_dispatched_at", "job_id", id, "error", err)
		}
	}
}

func (s *Service) MarkStaleWorkers(ctx context.Context) {
	query := `
		UPDATE workers
		SET status = 'STOPPED'
		WHERE status != 'STOPPED' 
		  AND last_heartbeat_at < NOW() - $1::interval
	`
	interval := s.heartbeatTimeout.String()
	cmd, err := s.pool.Exec(ctx, query, interval)
	if err != nil {
		s.logger.Error("Failed to mark stale workers", "error", err)
		return
	}

	if stopped := cmd.RowsAffected(); stopped > 0 {
		s.logger.Warn("Marked workers as STOPPED due to missed heartbeats", "count", stopped)
	}
}
