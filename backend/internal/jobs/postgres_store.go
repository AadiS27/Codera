package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/codera/code-executor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresJobStore struct {
	pool *pgxpool.Pool
}

func NewPostgresJobStore(pool *pgxpool.Pool) *PostgresJobStore {
	return &PostgresJobStore{
		pool: pool,
	}
}

func (s *PostgresJobStore) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *PostgresJobStore) Create(ctx context.Context, job *ExecutionJob) error {
	if job.Status != StatusQueued {
		return ErrInvalidState
	}

	query := `
		INSERT INTO executions (
			id, language, source_code, input, job_status, created_at, max_attempts
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	// Default max attempts is 5, but we can set it if provided on the job struct. 
	// For now let's default to 5 in code to be explicit.
	maxAttempts := job.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	_, err := s.pool.Exec(ctx, query,
		job.ID,
		string(job.Language),
		job.SourceCode,
		job.Input,
		string(job.Status),
		job.CreatedAt,
		maxAttempts,
	)

	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}
	return nil
}

func (s *PostgresJobStore) Get(ctx context.Context, id string) (*ExecutionJob, error) {
	query := `
		SELECT 
			id, language, source_code, input, job_status, result_status,
			stdout, stderr, exit_code, created_at, claimed_at, started_at, completed_at,
			worker_id, attempt_count, max_attempts, next_retry_at, lease_expires_at, 
			last_error, dead_lettered_at, last_dispatched_at
		FROM executions
		WHERE id = $1`

	row := s.pool.QueryRow(ctx, query, id)

	var job ExecutionJob
	var status string
	var resultStatus, stdout, stderr *string
	var exitCode *int

	err := row.Scan(
		&job.ID,
		&job.Language,
		&job.SourceCode,
		&job.Input,
		&status,
		&resultStatus,
		&stdout,
		&stderr,
		&exitCode,
		&job.CreatedAt,
		&job.ClaimedAt,
		&job.StartedAt,
		&job.CompletedAt,
		&job.WorkerID,
		&job.AttemptCount,
		&job.MaxAttempts,
		&job.NextRetryAt,
		&job.LeaseExpiresAt,
		&job.LastError,
		&job.DeadLetteredAt,
		&job.LastDispatchedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to scan job: %w", err)
	}

	job.Status = JobStatus(status)

	// Reconstruct the result if completed
	if resultStatus != nil {
		res := domain.ExecutionResult{
			Status: domain.ExecutionStatus(*resultStatus),
		}
		if stdout != nil {
			res.Stdout = *stdout
		}
		if stderr != nil {
			res.Stderr = *stderr
		}
		if exitCode != nil {
			res.ExitCode = *exitCode
		}
		job.Result = &res
	}

	return &job, nil
}

func (s *PostgresJobStore) Claim(ctx context.Context, id string, workerID string, leaseDuration time.Duration) error {
	query := `
		UPDATE executions 
		SET 
			job_status = $1, 
			worker_id = $2, 
			claimed_at = NOW(), 
			lease_expires_at = NOW() + $3::interval,
			attempt_count = attempt_count + 1
		WHERE 
			id = $4 
			AND job_status = $5
			AND (next_retry_at IS NULL OR next_retry_at <= NOW())`

	leaseInterval := fmt.Sprintf("%f seconds", leaseDuration.Seconds())
	cmd, err := s.pool.Exec(ctx, query, string(StatusClaimed), workerID, leaseInterval, id, string(StatusQueued))
	if err != nil {
		return fmt.Errorf("failed to claim job: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrInvalidState
	}

	return nil
}

func (s *PostgresJobStore) MarkRunning(ctx context.Context, id string, workerID string) error {
	query := `
		UPDATE executions 
		SET job_status = $1, started_at = NOW()
		WHERE id = $2 AND job_status = $3 AND worker_id = $4`

	cmd, err := s.pool.Exec(ctx, query, string(StatusRunning), id, string(StatusClaimed), workerID)
	if err != nil {
		return fmt.Errorf("failed to mark running: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrInvalidState
	}

	return nil
}

func (s *PostgresJobStore) Complete(ctx context.Context, id string, workerID string, result domain.ExecutionResult) error {
	query := `
		UPDATE executions
		SET 
			job_status = $1, 
			result_status = $2,
			stdout = $3,
			stderr = $4,
			exit_code = $5,
			completed_at = NOW(),
			worker_id = NULL,
			lease_expires_at = NULL
		WHERE id = $6 AND job_status = $7 AND worker_id = $8`

	cmd, err := s.pool.Exec(ctx, query,
		string(StatusCompleted),
		string(result.Status),
		result.Stdout,
		result.Stderr,
		result.ExitCode,
		id,
		string(StatusRunning),
		workerID,
	)

	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrInvalidState
	}

	return nil
}

func (s *PostgresJobStore) FailJob(ctx context.Context, id string, workerID string, errMsg string, backoff time.Duration) error {
	// Transition to either QUEUED (with retry) or DEAD_LETTERED
	query := `
		UPDATE executions
		SET 
			job_status = CASE 
				WHEN attempt_count < max_attempts THEN $1::text 
				ELSE $2::text 
			END,
			worker_id = NULL,
			lease_expires_at = NULL,
			last_error = $3,
			next_retry_at = CASE
				WHEN attempt_count < max_attempts THEN NOW() + $4::interval
				ELSE NULL
			END,
			dead_lettered_at = CASE
				WHEN attempt_count >= max_attempts THEN NOW()
				ELSE NULL
			END
		WHERE id = $5 AND worker_id = $6 AND job_status IN ($7, $8)`

	backoffInterval := fmt.Sprintf("%f seconds", backoff.Seconds())
	cmd, err := s.pool.Exec(ctx, query,
		string(StatusQueued),
		string(StatusDeadLettered),
		errMsg,
		backoffInterval,
		id,
		workerID,
		string(StatusClaimed),
		string(StatusRunning),
	)

	if err != nil {
		return fmt.Errorf("failed to fail job: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrInvalidState
	}

	return nil
}

func (s *PostgresJobStore) RenewLease(ctx context.Context, id string, workerID string, leaseDuration time.Duration) error {
	query := `
		UPDATE executions
		SET lease_expires_at = NOW() + $1::interval
		WHERE id = $2 AND job_status IN ($3, $4) AND worker_id = $5`

	leaseInterval := fmt.Sprintf("%f seconds", leaseDuration.Seconds())
	cmd, err := s.pool.Exec(ctx, query, leaseInterval, id, string(StatusClaimed), string(StatusRunning), workerID)
	
	if err != nil {
		return fmt.Errorf("failed to renew lease: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrInvalidState // Job might be completed, or ownership lost
	}

	return nil
}

func (s *PostgresJobStore) FindQueued(ctx context.Context, limit int) ([]string, error) {
	query := `
		SELECT id 
		FROM executions 
		WHERE job_status = $1 
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $2`

	rows, err := s.pool.Query(ctx, query, string(StatusQueued), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query queued jobs: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *PostgresJobStore) RecoverInterruptedRunning(ctx context.Context) (int64, error) {
	// This method might be obsolete in Phase 7 because we rely on lease expiry, but keep for legacy.
	return 0, nil
}

func (s *PostgresJobStore) RecoverExpiredLeases(ctx context.Context) (int64, error) {
	// Will be replaced by AbandonedJobs recovery in internal/recovery/
	return 0, nil
}
