-- +goose Up
CREATE TABLE IF NOT EXISTS files_meta (
	id TEXT PRIMARY KEY,
	total_parts INTEGER NOT NULL,
	payload JSONB NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS files_meta;
