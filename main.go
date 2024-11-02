package main

import (
	"basketball/config"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	// "github.com/mattn/go-sqlite3"
)

func main() {
	setupDatabase()
	runMigrations()
}

func setupDatabase() {
	if _, err := os.Stat(config.DatabaseFile); os.IsNotExist(err) {
		log.Println("Database file not found. Creating a new database.")
		file, err := os.Create(config.DatabaseFile)
		if err != nil {
			log.Fatalf("Failed to create database file: %v", err)
		}
		file.Close()
	}
}

func runMigrations() {
	m, err := migrate.New(
		"file://db/migrations",
		"sqlite3://"+config.DatabaseFile,
	)
	if err != nil {
		log.Fatalf("Failed to initialize migrations: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to apply migrations: %v", err)
	}
	log.Println("Migrations applied successfully.")
}
