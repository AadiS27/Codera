ALTER TABLE executions ADD COLUMN input TEXT NOT NULL DEFAULT '';
ALTER TABLE executions ADD COLUMN result_status TEXT;
ALTER TABLE executions ADD COLUMN stdout TEXT;
ALTER TABLE executions ADD COLUMN stderr TEXT;
ALTER TABLE executions ADD COLUMN exit_code INTEGER;

ALTER TABLE executions DROP COLUMN inputs;
ALTER TABLE executions DROP COLUMN results;
