package main

import (
	"basketball/config"
	"basketball/nba"

	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	config.LoadConfig()
	setupDatabase()
	runMigrations()

	if err := validateMigrations(); err != nil {
		log.Fatalf("Migration validation failed: %v", err)
	}

	scrapeCommonAllPlayers()
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

func scrapeCommonAllPlayers() {
	players := nba.CommonAllPlayers()
	insertPlayers(players)
}

func insertPlayers(players []nba.CommonAllPlayer) {
	db, err := sql.Open("sqlite3", config.DatabaseFile)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error beginning transaction: %v", err)
	}

	stmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO players (
			id,
			name,
			team_id
			) VALUES (?, ?, ?)`,
	)
	if err != nil {
		tx.Rollback()
		log.Printf("Error preparing statement: %v", err)
	}
	defer stmt.Close()

	for _, player := range players {
		res, err := stmt.Exec(
			player.ID,
			player.Name,
			player.TeamID,
		)
		if err != nil {
			tx.Rollback()
			log.Printf("Error inserting player %s(%d): %v", player.Name, player.ID, err)
			return
		}
		lastId, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			log.Printf("Failed to get last insert ID for player %s (%d): %v", player.Name, player.ID, err)
			return
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			tx.Rollback()
			log.Printf("Failed to get rows affected for player %s (%d): %v", player.Name, player.ID, err)
			return
		}
		log.Printf("Processed player %s (ID: %d) with TeamID: %d. Rows affected: %d. Last insert Id: %d",
			player.Name,
			player.ID,
			player.TeamID,
			rowsAffected,
			lastId,
		)
	}
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v", err)
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

	if count != 31 {
		return fmt.Errorf("expected 31 teams, found %d", count)
	}

	var name string
	err = db.QueryRow("SELECT name FROM teams WHERE id = 1610612752").Scan(&name)
	if err != nil {
		return fmt.Errorf("failed to find Knicks: %v", err)
	}
	if name != "New York Knicks" {
		return fmt.Errorf("expected team.id 1610612752 to have name 'New York Knicks', got '%s'", name)
	}
	err = db.QueryRow("SELECT name FROM teams WHERE id = 0").Scan(&name)
	if err != nil {
		return fmt.Errorf("failed to find NULL_TEAM: %v", err)
	}
	if name != "NULL_TEAM" {
		return fmt.Errorf("expected team.id 0 to have name 'NULL_TEAM', got '%s'", name)
	}
	log.Printf("Database validation successful: found %d teams\n", count)
	return nil
}
