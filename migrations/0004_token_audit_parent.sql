-- +goose Up
ALTER TABLE tokens ADD COLUMN parent_token_hash TEXT;

CREATE TABLE IF NOT EXISTS token_audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  action TEXT NOT NULL,
  token_hash TEXT,
  owner_user_id INTEGER,
  actor_user_id INTEGER,
  parent_token_hash TEXT,
  meta TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS token_audit;
-- Note: SQLite doesn't support dropping columns easily; parent_token_hash will remain if downgrading.
