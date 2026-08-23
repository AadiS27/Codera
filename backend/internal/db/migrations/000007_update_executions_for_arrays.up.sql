ALTER TABLE executions ADD COLUMN inputs JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE executions ADD COLUMN results JSONB;

-- Migrate data if necessary (we can just leave old data as is or drop it)
-- Since this is development, we can just drop the old columns
ALTER TABLE executions DROP COLUMN input;
ALTER TABLE executions DROP COLUMN result_status;
ALTER TABLE executions DROP COLUMN stdout;
ALTER TABLE executions DROP COLUMN stderr;
ALTER TABLE executions DROP COLUMN exit_code;
