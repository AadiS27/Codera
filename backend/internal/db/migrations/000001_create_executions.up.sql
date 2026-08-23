CREATE TABLE executions (
    id TEXT PRIMARY KEY,
    language TEXT NOT NULL,
    source_code TEXT NOT NULL,
    input TEXT NOT NULL DEFAULT '',

    job_status TEXT NOT NULL CHECK (job_status IN ('QUEUED', 'RUNNING', 'COMPLETED')),
    result_status TEXT,

    stdout TEXT,
    stderr TEXT,
    exit_code INTEGER,

    created_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

-- Index for the Reconciler (finding QUEUED jobs ordered by creation time)
CREATE INDEX idx_executions_job_status_created_at ON executions (job_status, created_at);
