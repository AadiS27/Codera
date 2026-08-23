package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/codera/code-executor/internal/queue"
)

// RunStartupRecovery handles any jobs stranded by a previous crash.
// 1. Force completes any RUNNING jobs (internal error).
// 2. Re-enqueues all QUEUED jobs into the memory queue.
func RunStartupRecovery(ctx context.Context, store *PostgresJobStore, q queue.JobQueue, logger *slog.Logger) error {
	logger.Info("Starting database recovery process")

	// 1. Handle stranded RUNNING jobs
	count, err := store.RecoverInterruptedRunning(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		logger.Warn("Recovered interrupted running jobs", "count", count)
	}

	// 2. Re-enqueue QUEUED jobs
	// We'll fetch them all in one go since on startup the in-memory queue is empty.
	// In a real huge scale system, we'd batch this.
	queued, err := store.FindQueued(ctx, 1000) 
	if err != nil {
		return err
	}

	enqueued := 0
	for _, id := range queued {
		if err := q.Enqueue(ctx, id); err != nil {
			logger.Error("Failed to enqueue job during recovery", "job_id", id, "error", err)
			continue
		}
		enqueued++
	}

	if enqueued > 0 {
		logger.Info("Recovered queued jobs", "count", enqueued)
	}

	logger.Info("Database recovery complete")
	return nil
}

// StartReconciler runs in the background to ensure no QUEUED jobs are lost due to crashes 
// between the Postgres COMMIT and Memory Queue ENQUEUE operations.
func StartReconciler(ctx context.Context, store *PostgresJobStore, q queue.JobQueue, logger *slog.Logger, interval time.Duration, batchSize int32) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("Background Reconciler started", "interval", interval, "batch_size", batchSize)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Background Reconciler stopped")
			return
		case <-ticker.C:
			// Fetch up to batchSize jobs
			jobs, err := store.FindQueued(ctx, int(batchSize))
			if err != nil {
				logger.Error("Reconciler failed to query queued jobs", "error", err)
				continue
			}

			enqueued := 0
			for _, id := range jobs {
				err := q.Enqueue(ctx, id)
				if err != nil {
					if err == queue.ErrQueueFull {
						// The queue is full, stop trying to shove more in for this tick
						break
					}
					// Ignore queue closed error here, as ctx.Done() will catch it next loop
				} else {
					enqueued++
				}
			}

			if enqueued > 0 {
				logger.Debug("Reconciler processed stranded jobs", "count", enqueued)
			}
		}
	}
}
