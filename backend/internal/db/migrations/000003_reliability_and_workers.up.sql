CREATE TABLE IF NOT EXISTS workers (
    worker_id TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE executions ADD COLUMN claimed_at TIMESTAMPTZ;
ALTER TABLE executions ADD COLUMN attempt_count INT NOT NULL DEFAULT 0;
ALTER TABLE executions ADD COLUMN max_attempts INT NOT NULL DEFAULT 5;
ALTER TABLE executions ADD COLUMN next_retry_at TIMESTAMPTZ;
ALTER TABLE executions ADD COLUMN last_error TEXT;
ALTER TABLE executions ADD COLUMN dead_lettered_at TIMESTAMPTZ;
ALTER TABLE executions ADD COLUMN last_dispatched_at TIMESTAMPTZ;

CREATE INDEX idx_executions_retry ON executions(job_status, next_retry_at);
CREATE INDEX idx_executions_lease ON executions(job_status, lease_expires_at);
CREATE INDEX idx_executions_worker ON executions(worker_id);
