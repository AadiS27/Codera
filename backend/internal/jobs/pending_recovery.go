package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/codera/code-executor/internal/queue"
	"github.com/redis/go-redis/v9"
)

// RecoverPendingMessages runs in the background and sweeps Redis for stale messages
// using XAUTOCLAIM. If it finds a stale message, it checks Postgres.
// If the job is COMPLETED, it ACKs the message.
// If the job is RUNNING, it ignores it (the lease recovery handles Postgres state).
// If the job is QUEUED, it leaves the message pending so a worker can consume it (or we can just leave it claimed for this instance).
// Wait, XAUTOCLAIM actually assigns ownership to this caller. So this process should probably
// just run as a worker role, and when it claims, it processes the job.
// But the plan says "RecoverPendingMessages: sweep stale messages... if COMPLETED, ACK. If RUNNING, drop message".
// Let's implement a standalone recovery loop.

type PendingRecovery struct {
	logger *slog.Logger
	redis  *redis.Client
	store  *PostgresJobStore
	queue  *queue.RedisQueue

	stream           string
	group            string
	idleTimeout      time.Duration
	claimBatchSize   int64
	recoveryInterval time.Duration
}

func NewPendingRecovery(logger *slog.Logger, r *redis.Client, s *PostgresJobStore, q *queue.RedisQueue, stream, group string, idle time.Duration, batch int64) *PendingRecovery {
	return &PendingRecovery{
		logger:           logger,
		redis:            r,
		store:            s,
		queue:            q,
		stream:           stream,
		group:            group,
		idleTimeout:      idle,
		claimBatchSize:   batch,
		recoveryInterval: 15 * time.Second, // Run recovery every 15 seconds
	}
}

func (pr *PendingRecovery) Start(ctx context.Context) {
	ticker := time.NewTicker(pr.recoveryInterval)
	defer ticker.Stop()

	pr.logger.Info("Pending message recovery started", "idle_timeout", pr.idleTimeout, "batch_size", pr.claimBatchSize)

	for {
		select {
		case <-ctx.Done():
			pr.logger.Info("Pending message recovery stopped")
			return
		case <-ticker.C:
			pr.runRecovery(ctx)
		}
	}
}

func (pr *PendingRecovery) runRecovery(ctx context.Context) {
	// We use a dummy consumer name for recovery since we are just inspecting/acking.
	// But XAUTOCLAIM requires a consumer name to claim it TO.
	// We don't actually want to claim it to execute it here, we just want to inspect it.
	// However, XAUTOCLAIM is the easiest way to atomically get old messages.
	recoveryConsumer := "system-recovery"

	start := "-"
	for {
		// XAUTOCLAIM syntax: XAUTOCLAIM stream group consumer min-idle-time start [COUNT count]
		args := &redis.XAutoClaimArgs{
			Stream:   pr.stream,
			Group:    pr.group,
			Consumer: recoveryConsumer,
			MinIdle:  pr.idleTimeout,
			Start:    start,
			Count:    pr.claimBatchSize,
		}

		msgs, nextStart, err := pr.redis.XAutoClaim(ctx, args).Result()
		if err != nil {
			pr.logger.Error("Failed to run XAUTOCLAIM", "error", err)
			return
		}

		for _, msg := range msgs {
			jobID, ok := msg.Values["job_id"].(string)
			if !ok {
				// Malformed message, ACK and delete
				pr.redis.XAck(ctx, pr.stream, pr.group, msg.ID)
				continue
			}

			// Check job state in Postgres
			job, err := pr.store.Get(ctx, jobID)
			if err != nil {
				if err == ErrJobNotFound {
					pr.logger.Warn("Pending message points to missing job, ACKing", "job_id", jobID)
					pr.redis.XAck(ctx, pr.stream, pr.group, msg.ID)
				} else {
					pr.logger.Error("Failed to fetch job during recovery", "job_id", jobID, "error", err)
				}
				continue
			}

			if job.Status == StatusCompleted {
				pr.logger.Info("Pending message belongs to COMPLETED job, ACKing", "job_id", jobID, "msg_id", msg.ID)
				pr.redis.XAck(ctx, pr.stream, pr.group, msg.ID)
			} else if job.Status == StatusRunning {
				// The job is running. Maybe the lease is valid, maybe not.
				// The Postgres lease recovery handles requeueing expired leases.
				// For the Redis message, we can just leave it claimed by 'system-recovery' (it will idle again),
				// or we can explicitly XACK it, knowing that if the job fails, Postgres recovery will XADD a NEW message.
				// Let's XACK it to prevent queue bloat, because Postgres will dispatch a new message if it recovers the lease.
				pr.logger.Debug("Pending message belongs to RUNNING job, ACKing as lease recovery will handle failures", "job_id", jobID)
				pr.redis.XAck(ctx, pr.stream, pr.group, msg.ID)
			} else if job.Status == StatusQueued {
				// Job is QUEUED, but message was pending. 
				// We can XACK this message and enqueue a new one to be safe, or just let XAUTOCLAIM deliver it?
				// Actually, if we XACK it and call Enqueue, it cleanly resets it for workers to consume.
				pr.logger.Info("Pending message belongs to QUEUED job, re-enqueuing", "job_id", jobID)
				pr.redis.XAck(ctx, pr.stream, pr.group, msg.ID)
				_ = pr.queue.Enqueue(ctx, jobID)
			}
		}

		if nextStart == "0-0" || len(msgs) == 0 {
			break
		}
		start = nextStart
	}
}
