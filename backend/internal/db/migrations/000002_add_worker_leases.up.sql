ALTER TABLE executions ADD COLUMN worker_id TEXT;
ALTER TABLE executions ADD COLUMN lease_expires_at TIMESTAMPTZ;
