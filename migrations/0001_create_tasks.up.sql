CREATE TABLE IF NOT EXISTS tasks (
	id BIGSERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	recurrence_type TEXT,
	recurrence_interval INT,
	recurrence_dates JSONB,
	recurrence_end_date TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);

ALTER TABLE IF EXISTS tasks ADD COLUMN IF NOT EXISTS recurrence_type TEXT;
ALTER TABLE IF EXISTS tasks ADD COLUMN IF NOT EXISTS recurrence_interval INT;
ALTER TABLE IF EXISTS tasks ADD COLUMN IF NOT EXISTS recurrence_dates JSONB;
ALTER TABLE IF EXISTS tasks ADD COLUMN IF NOT EXISTS recurrence_end_date TIMESTAMPTZ;