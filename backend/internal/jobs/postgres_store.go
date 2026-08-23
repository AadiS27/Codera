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

func (s *PostgresJobStore) Create(ctx context.Context, job *ExecutionJob) error {
	if job.Status != StatusQueued {
		return ErrInvalidState
	}

	query := `
		INSERT INTO executions (
			id, language, source_code, input, job_status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)`

	_, err := s.pool.Exec(ctx, query,
		job.ID,
		job.Language,
		job.SourceCode,
		job.Input,
		string(job.Status),
		job.CreatedAt,
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
			stdout, stderr, exit_code, created_at, started_at, completed_at
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
		&job.StartedAt,
		&job.CompletedAt,
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

func (s *PostgresJobStore) MarkRunning(ctx context.Context, id string, workerID string, leaseDuration time.Duration) error {
	query := `
		UPDATE executions 
		SET job_status = $1, started_at = NOW(), worker_id = $2, lease_expires_at = NOW() + $3::interval
		WHERE id = $4 AND job_status = $5`

	leaseInterval := fmt.Sprintf("%f seconds", leaseDuration.Seconds())
	cmd, err := s.pool.Exec(ctx, query, string(StatusRunning), workerID, leaseInterval, id, string(StatusQueued))
	if err != nil {
		return fmt.Errorf("failed to mark running: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		// Could mean it doesn't exist, or it's not QUEUED anymore.
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

// FindQueued returns up to `limit` jobs that are currently QUEUED, ordered by creation time.
func (s *PostgresJobStore) FindQueued(ctx context.Context, limit int) ([]string, error) {
	query := `
		SELECT id 
		FROM executions 
		WHERE job_status = $1
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

// RecoverInterruptedRunning finds any RUNNING jobs and forcefully completes them with an INTERNAL_ERROR.
func (s *PostgresJobStore) RecoverInterruptedRunning(ctx context.Context) (int64, error) {
	query := `
		UPDATE executions
		SET 
			job_status = $1,
			result_status = $2,
			stderr = $3,
			exit_code = -1,
			completed_at = NOW()
		WHERE job_status = $4`

	cmd, err := s.pool.Exec(ctx, query,
		string(StatusCompleted),
		string(domain.StatusRuntimeError), // Map platform error to Runtime Error for now
		"Internal Platform Error: Job interrupted during shutdown",
		string(StatusRunning),
	)

	if err != nil {
		return 0, fmt.Errorf("failed to recover interrupted jobs: %w", err)
	}

	return cmd.RowsAffected(), nil
}
// RenewLease extends the lease for a running job owned by a specific worker.
func (s *PostgresJobStore) RenewLease(ctx context.Context, id string, workerID string, leaseDuration time.Duration) error {
	query := `
		UPDATE executions
		SET lease_expires_at = NOW() + $1::interval
		WHERE id = $2 AND job_status = $3 AND worker_id = $4`

	leaseInterval := fmt.Sprintf("%f seconds", leaseDuration.Seconds())
	cmd, err := s.pool.Exec(ctx, query, leaseInterval, id, string(StatusRunning), workerID)
	
	if err != nil {
		return fmt.Errorf("failed to renew lease: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrInvalidState // Job might be completed, or ownership lost
	}

	return nil
}

// RecoverExpiredLeases finds RUNNING jobs with expired leases and resets them to QUEUED for redispatch.
// This is the At-Least-Once safety net for worker crashes without pending redis message recovery.
func (s *PostgresJobStore) RecoverExpiredLeases(ctx context.Context) (int64, error) {
	query := `
		UPDATE executions
		SET 
			job_status = $1,
			worker_id = NULL,
			lease_expires_at = NULL
		WHERE job_status = $2 AND lease_expires_at < NOW()`

	cmd, err := s.pool.Exec(ctx, query, string(StatusQueued), string(StatusRunning))
	if err != nil {
		return 0, fmt.Errorf("failed to recover expired leases: %w", err)
	}

	return cmd.RowsAffected(), nil
}
