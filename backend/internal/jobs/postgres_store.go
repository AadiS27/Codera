package jobs

import (
	"context"
	"errors"
	"fmt"

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

func (s *PostgresJobStore) MarkRunning(ctx context.Context, id string) error {
	query := `
		UPDATE executions 
		SET job_status = $1, started_at = NOW()
		WHERE id = $2 AND job_status = $3`

	cmd, err := s.pool.Exec(ctx, query, string(StatusRunning), id, string(StatusQueued))
	if err != nil {
		return fmt.Errorf("failed to mark running: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		// Could mean it doesn't exist, or it's not QUEUED anymore.
		return ErrInvalidState
	}

	return nil
}

func (s *PostgresJobStore) Complete(ctx context.Context, id string, result domain.ExecutionResult) error {
	query := `
		UPDATE executions
		SET 
			job_status = $1, 
			result_status = $2,
			stdout = $3,
			stderr = $4,
			exit_code = $5,
			completed_at = NOW()
		WHERE id = $6 AND job_status = $7`

	cmd, err := s.pool.Exec(ctx, query,
		string(StatusCompleted),
		string(result.Status),
		result.Stdout,
		result.Stderr,
		result.ExitCode,
		id,
		string(StatusRunning),
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
