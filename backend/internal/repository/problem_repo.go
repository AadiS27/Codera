package repository

import (
	"context"
	"errors"

	"github.com/codera/code-executor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProblemNotFound = errors.New("problem not found")

type ProblemRepository interface {
	Create(ctx context.Context, p *domain.Problem) error
	GetByID(ctx context.Context, id string) (domain.Problem, error)
	// Additional methods like Update, List can be added later
}

type PostgresProblemRepository struct {
	db *pgxpool.Pool
}

func NewPostgresProblemRepository(db *pgxpool.Pool) *PostgresProblemRepository {
	return &PostgresProblemRepository{db: db}
}

func (r *PostgresProblemRepository) Create(ctx context.Context, p *domain.Problem) error {
	query := `
		INSERT INTO problems (
			title, slug, description, input_description, output_description,
			constraints, time_limit_ms, memory_limit_mb, comparison_mode,
			float_epsilon, status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id
	`
	err := r.db.QueryRow(ctx, query,
		p.Title, p.Slug, p.Description, p.InputDescription, p.OutputDescription,
		p.Constraints, p.TimeLimitMs, p.MemoryLimitMB, string(p.ComparisonMode),
		p.FloatEpsilon, string(p.Status), p.CreatedAt,
	).Scan(&p.ID)
	return err
}

func (r *PostgresProblemRepository) GetByID(ctx context.Context, id string) (domain.Problem, error) {
	query := `
		SELECT id, title, slug, description, input_description, output_description,
		constraints, time_limit_ms, memory_limit_mb, comparison_mode,
		float_epsilon, status, created_at
		FROM problems WHERE id = $1
	`
	var p domain.Problem
	var compMode string
	var status string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Title, &p.Slug, &p.Description, &p.InputDescription, &p.OutputDescription,
		&p.Constraints, &p.TimeLimitMs, &p.MemoryLimitMB, &compMode,
		&p.FloatEpsilon, &status, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Problem{}, ErrProblemNotFound
		}
		return domain.Problem{}, err
	}

	p.ComparisonMode = domain.ComparisonMode(compMode)
	p.Status = domain.ProblemStatus(status)

	return p, nil
}
