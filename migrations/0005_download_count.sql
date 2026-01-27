-- +goose Up
ALTER TABLE packages ADD COLUMN download_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE package_versions ADD COLUMN download_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- Note: SQLite doesn't support dropping columns easily; download_count will remain if downgrading.
