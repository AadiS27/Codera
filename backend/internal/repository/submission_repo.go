package repository

import (
	"context"
	"errors"

	"github.com/codera/code-executor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSubmissionNotFound = errors.New("submission not found")

type SubmissionRepository interface {
	Create(ctx context.Context, s *domain.Submission) error
	GetByID(ctx context.Context, id string) (domain.Submission, error)
	Update(ctx context.Context, s domain.Submission) error
}

type PostgresSubmissionRepository struct {
	db *pgxpool.Pool
}

func NewPostgresSubmissionRepository(db *pgxpool.Pool) *PostgresSubmissionRepository {
	return &PostgresSubmissionRepository{db: db}
}

func (r *PostgresSubmissionRepository) Create(ctx context.Context, s *domain.Submission) error {
	query := `
		INSERT INTO submissions (
			user_id, problem_id, language, source_code, status, verdict,
			execution_time_ms, memory_used_bytes, passed_test_cases, total_test_cases,
			created_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id
	`
	err := r.db.QueryRow(ctx, query,
		s.UserID, s.ProblemID, string(s.Language), s.SourceCode, string(s.Status), string(s.Verdict),
		s.ExecutionTimeMs, s.MemoryUsedBytes, s.PassedTestCases, s.TotalTestCases,
		s.CreatedAt, s.CompletedAt,
	).Scan(&s.ID)
	return err
}

func (r *PostgresSubmissionRepository) GetByID(ctx context.Context, id string) (domain.Submission, error) {
	query := `
		SELECT id, user_id, problem_id, language, source_code, status, verdict,
		execution_time_ms, memory_used_bytes, passed_test_cases, total_test_cases,
		created_at, completed_at
		FROM submissions WHERE id = $1
	`
	var s domain.Submission
	var lang, status, verdict string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.UserID, &s.ProblemID, &lang, &s.SourceCode, &status, &verdict,
		&s.ExecutionTimeMs, &s.MemoryUsedBytes, &s.PassedTestCases, &s.TotalTestCases,
		&s.CreatedAt, &s.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Submission{}, ErrSubmissionNotFound
		}
		return domain.Submission{}, err
	}

	s.Language = domain.Language(lang)
	s.Status = domain.SubmissionStatus(status)
	s.Verdict = domain.Verdict(verdict)

	return s, nil
}

func (r *PostgresSubmissionRepository) Update(ctx context.Context, s domain.Submission) error {
	query := `
		UPDATE submissions SET
			status = $1, verdict = $2, execution_time_ms = $3,
			memory_used_bytes = $4, passed_test_cases = $5, total_test_cases = $6,
			completed_at = $7
		WHERE id = $8
	`
	_, err := r.db.Exec(ctx, query,
		string(s.Status), string(s.Verdict), s.ExecutionTimeMs,
		s.MemoryUsedBytes, s.PassedTestCases, s.TotalTestCases,
		s.CompletedAt, s.ID,
	)
	return err
}
