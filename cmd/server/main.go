package main

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"ebuild/internal/api"
)

func main() {
	dbPath := os.Getenv("DEV_DB")
	if dbPath == "" {
		dbPath = "dev.db"
	}

	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	signingKey := []byte("dev-signing-key")

	r := api.SetupRouter(db, signingKey)

	log.Println("starting server on :8080")
	r.Run(":8080")
}
