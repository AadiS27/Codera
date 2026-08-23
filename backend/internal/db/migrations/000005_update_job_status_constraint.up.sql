ALTER TABLE executions DROP CONSTRAINT IF EXISTS executions_job_status_check;
ALTER TABLE executions ADD CONSTRAINT executions_job_status_check CHECK (job_status IN ('QUEUED', 'CLAIMED', 'RUNNING', 'COMPLETED', 'DEAD_LETTERED'));
