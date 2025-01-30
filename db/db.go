package db

import (
	"basketball/config"
	"basketball/nba"

	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func SetupDatabase() {
	if _, err := os.Stat(config.DatabaseFile); os.IsNotExist(err) {
		log.Println("Database file not found. Creating a new database.")
		file, err := os.Create(config.DatabaseFile)
		if err != nil {
			log.Fatalf("Failed to create database file: %v", err)
		}
		file.Close()
	}
}

func RunMigrations() {
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

func ValidateMigrations() error {
	db, err := sql.Open("sqlite3", config.DatabaseFile)
	if err != nil {
		return fmt.Errorf("failed to open databse: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("Select COUNT(*) FROM teams").Scan(&count)
	if err != nil {
		panic(fmt.Errorf("failed to query teams table: %v", err))
	}

	if count != 31 {
		panic(fmt.Errorf("expected 31 teams, found %d", count))
	}

	var name string
	err = db.QueryRow("SELECT name FROM teams WHERE id = 1610612752").Scan(&name)
	if err != nil {
		panic(fmt.Errorf("failed to find Knicks: %v", err))
	}
	if name != "New York Knicks" {
		panic(fmt.Errorf("expected team.id 1610612752 to have name 'New York Knicks', got '%s'", name))
	}
	err = db.QueryRow("SELECT name FROM teams WHERE id = 0").Scan(&name)
	if err != nil {
		panic(fmt.Errorf("failed to find NULL_TEAM: %v", err))
	}
	if name != "NULL_TEAM" {
		panic(fmt.Errorf("expected team.id 0 to have name 'NULL_TEAM', got '%s'", name))
	}
	log.Printf("Database validation successful: found %d teams\n", count)
	return nil
}

//go:embed asciitball.txt
var chunkyDunker string

func PlayerIDFromCode(playerCode string) (int, error) {
	db, err := sql.Open("sqlite3", config.DatabaseFile)
	if err != nil {
		return -1, fmt.Errorf("failed to open databse: %v", err)
	}
	defer db.Close()

	var id int
	err = db.QueryRow("SELECT id FROM players WHERE name = $1", playerCode).Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func InsertPlayers(players []nba.CommonAllPlayer) {
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
		if player.PersonID == nil {
			log.Printf("found player with nil PersonID: %s", *player.DisplayFirstLast)
			continue
		}
		if player.DisplayFirstLast == nil {
			log.Printf("found player with nil DisplayFirstLast: %d", int(*player.PersonID))
			continue
		}
		if player.TeamID == nil {
			log.Printf("found player with nil TeamID: %s", *player.DisplayFirstLast)
			continue
		}
		_, err := stmt.Exec(
			*player.PersonID,
			*player.DisplayFirstLast,
			*player.TeamID,
		)
		if err != nil {
			tx.Rollback()
			log.Printf("error inserting player %s(%d): %v\n", *player.DisplayFirstLast, int(*player.PersonID), err)
			return
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v", err)
	}
}
