package main

import (
	"basketball/config"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	// "github.com/mattn/go-sqlite3"
)

func main() {
	config.LoadConfig()
	setupDatabase()
	runMigrations()

	if err := validateMigrations(); err != nil {
		log.Fatalf("Migration validation failed: %v", err)
	}
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

func validateMigrations() error {
	db, err := sql.Open("sqlite3", config.DatabaseFile)
	if err != nil {
		return fmt.Errorf("failed to open databse: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("Select COUNT(*) FROM teams").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query teams table: %v", err)
	}

	if count != 30 {
		return fmt.Errorf("expected 30 teams, found %d", count)
	}

	var name string
	err = db.QueryRow("SELECT name FROM teams WHERE team_id = 1610612752").Scan(&name)
	if err != nil {
		return fmt.Errorf("failed to find Knicks: %v", err)
	}
	if name != "New York Knicks" {
		return fmt.Errorf("expected 'New York Knicks', got '%s'", name)
	}
	log.Printf("Database validation successful: found %d teams\n", count)
	return nil
}
