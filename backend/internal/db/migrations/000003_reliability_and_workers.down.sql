DROP INDEX IF EXISTS idx_executions_retry;
DROP INDEX IF EXISTS idx_executions_lease;
DROP INDEX IF EXISTS idx_executions_worker;

ALTER TABLE executions DROP COLUMN claimed_at;
ALTER TABLE executions DROP COLUMN attempt_count;
ALTER TABLE executions DROP COLUMN max_attempts;
ALTER TABLE executions DROP COLUMN next_retry_at;
ALTER TABLE executions DROP COLUMN last_error;
ALTER TABLE executions DROP COLUMN dead_lettered_at;
ALTER TABLE executions DROP COLUMN last_dispatched_at;

DROP TABLE IF EXISTS workers;
