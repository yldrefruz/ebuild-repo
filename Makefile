# Simple Makefile for common dev tasks


.PHONY: sqlc-gen install-goose migrate-up migrate-down run-dev build

# Generate typed DB access using sqlc (requires sqlc installed). Currently not used.
sqlc-gen:
	@echo "running sqlc generate..."
	@sqlc generate


# Install pressly/goose migration CLI
install-goose:
	@echo "installing pressly/goose migration CLI..."
	@CGO_ENABLED=1 go install github.com/pressly/goose/v3/cmd/goose@latest || \
	  (echo "go install goose failed; ensure gcc and go toolchain available");

# Run migrations up against the local SQLite dev DB
# TODO: For production, this must run against Postgres with appropriate connection string
# Uses sqlite file with foreign keys enabled

migrate-up:
	@echo "running migrations up against ./dev.db using goose"
	@echo 'GOPATH env is' `go env GOPATH`
	@if command -v `go env GOPATH`/bin/goose >/dev/null 2>&1; then \
		`go env GOPATH`/bin/goose -dir migrations sqlite3 dev.db up; \
	else \
		echo "goose not found; run 'make install-goose' or install goose manually"; exit 1; \
	fi


# Rollback last migration (down)
migrate-down:
	@echo "rolling back last migration against ./dev.db using goose"
	@if command -v `go env GOPATH`/bin/goose >/dev/null 2>&1; then \
		`go env GOPATH`/bin/goose -dir migrations sqlite3 dev.db down; \
	else \
		echo "goose not found; run 'make install-goose' or install goose manually"; exit 1; \
	fi

run-dev:
	@echo "starting dev server"
	@go run ./cmd/server

build:
	@echo "building ebuild server to bin/ebuild"
	@mkdir -p bin
	@go build -o bin/ebuild ./cmd/server
