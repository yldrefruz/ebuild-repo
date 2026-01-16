-- +goose Up
CREATE TABLE IF NOT EXISTS package_maintainers (
  package_id INTEGER NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  PRIMARY KEY (package_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS package_maintainers;
