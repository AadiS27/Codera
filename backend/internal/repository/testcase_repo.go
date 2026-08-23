package repository

import (
	"context"

	"github.com/codera/code-executor/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TestCaseRepository interface {
	Create(ctx context.Context, tc domain.TestCase) error
	GetByProblemID(ctx context.Context, problemID string, includeHidden bool) ([]domain.TestCase, error)
}

type PostgresTestCaseRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTestCaseRepository(db *pgxpool.Pool) *PostgresTestCaseRepository {
	return &PostgresTestCaseRepository{db: db}
}

func (r *PostgresTestCaseRepository) Create(ctx context.Context, tc domain.TestCase) error {
	query := `
		INSERT INTO test_cases (
			id, problem_id, input, expected_output, visibility, sort_order, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`
	_, err := r.db.Exec(ctx, query,
		tc.ID, tc.ProblemID, tc.Input, tc.ExpectedOutput, string(tc.Visibility),
		tc.SortOrder, tc.CreatedAt,
	)
	return err
}

func (r *PostgresTestCaseRepository) GetByProblemID(ctx context.Context, problemID string, includeHidden bool) ([]domain.TestCase, error) {
	query := `
		SELECT id, problem_id, input, expected_output, visibility, sort_order, created_at
		FROM test_cases
		WHERE problem_id = $1
	`
	args := []interface{}{problemID}
	
	if !includeHidden {
		query += ` AND visibility = 'PUBLIC'`
	}
	
	query += ` ORDER BY sort_order ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []domain.TestCase
	for rows.Next() {
		var tc domain.TestCase
		var visibility string
		if err := rows.Scan(
			&tc.ID, &tc.ProblemID, &tc.Input, &tc.ExpectedOutput, &visibility,
			&tc.SortOrder, &tc.CreatedAt,
		); err != nil {
			return nil, err
		}
		tc.Visibility = domain.TestCaseVisibility(visibility)
		testCases = append(testCases, tc)
	}

	return testCases, rows.Err()
}
