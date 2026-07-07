package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://mapaturbo:mapaturbo_password@localhost:5432/mapaturbo?sslmode=disable"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("goose: failed to open DB: %v\n", err)
	}
	defer db.Close()

	migrationsDir := "db/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = "../db/migrations"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			migrationsDir = "../../db/migrations"
		}
	}

	log.Printf("Applying migrations from: %s\n", migrationsDir)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose: failed to set dialect: %v\n", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		log.Fatalf("goose: migration failed: %v\n", err)
	}
	log.Println("Goose migrations applied successfully!")
}
