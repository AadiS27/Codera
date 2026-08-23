package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkerStatus string

const (
	StatusStarting WorkerStatus = "STARTING"
	StatusActive   WorkerStatus = "ACTIVE"
	StatusDraining WorkerStatus = "DRAINING"
	StatusStopped  WorkerStatus = "STOPPED"
)

// Heartbeat handles registering the worker and emitting periodic heartbeats.
type Heartbeat struct {
	pool       *pgxpool.Pool
	logger     *slog.Logger
	workerID   string
	interval   time.Duration
}

func NewHeartbeat(pool *pgxpool.Pool, logger *slog.Logger, workerID string, interval time.Duration) *Heartbeat {
	return &Heartbeat{
		pool:     pool,
		logger:   logger,
		workerID: workerID,
		interval: interval,
	}
}

func (h *Heartbeat) UpdateStatus(ctx context.Context, status WorkerStatus) error {
	query := `
		INSERT INTO workers (worker_id, status, started_at, last_heartbeat_at, updated_at)
		VALUES ($1, $2, NOW(), NOW(), NOW())
		ON CONFLICT (worker_id) 
		DO UPDATE SET 
			status = $2, 
			last_heartbeat_at = NOW(), 
			updated_at = NOW()
	`
	_, err := h.pool.Exec(ctx, query, h.workerID, string(status))
	return err
}

func (h *Heartbeat) Run(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			query := `
				UPDATE workers 
				SET last_heartbeat_at = NOW(), updated_at = NOW() 
				WHERE worker_id = $1
			`
			_, err := h.pool.Exec(context.Background(), query, h.workerID)
			if err != nil {
				h.logger.Error("Failed to update worker heartbeat", "error", err)
			}
		}
	}
}
