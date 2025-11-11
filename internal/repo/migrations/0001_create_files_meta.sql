-- +goose Up
CREATE TABLE IF NOT EXISTS files_meta (
	id TEXT PRIMARY KEY,
	file_name TEXT NOT NULL,
	total_parts INTEGER NOT NULL,
	size BIGINT NOT NULL,
	parts JSONB NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS files_meta;
