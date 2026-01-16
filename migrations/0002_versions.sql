-- +goose Up
CREATE TABLE IF NOT EXISTS package_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  package_id INTEGER NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
  version TEXT NOT NULL,
  metadata TEXT,
  released_by INTEGER NOT NULL REFERENCES users(id) ON DELETE SET NULL,
  released_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  is_deprecated INTEGER NOT NULL DEFAULT 0,
  UNIQUE(package_id, version)
);

CREATE TABLE IF NOT EXISTS artifacts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  package_version_id INTEGER NOT NULL REFERENCES package_versions(id) ON DELETE CASCADE,
  blob_url TEXT NOT NULL,
  filename TEXT,
  size_bytes INTEGER,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS package_versions;
