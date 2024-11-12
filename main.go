package main

import (
	"basketball/config"
	"basketball/nba"
	"errors"
	"strings"
	"time"

	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	flag "github.com/spf13/pflag"
)

var statlinePlayerName string

func init() {
	flag.StringVarP(&statlinePlayerName, "statline", "s", "", "player name to get statline for")
	flag.Parse()
}

func main() {
	config.LoadConfig()
	setupDatabase()
	runMigrations()

	if err := validateMigrations(); err != nil {
		log.Fatalf("Migration validation failed: %v", err)
	}

	scrapeCommonAllPlayers()

	if len(statlinePlayerName) != 0 {
		statline(statlinePlayerName)
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
		if _, err := res.LastInsertId(); err != nil {
			tx.Rollback()
			log.Printf("Failed to get last insert ID for player %s (%d): %v", player.Name, player.ID, err)
			return
		}
		if _, err := res.RowsAffected(); err != nil {
			tx.Rollback()
			log.Printf("Failed to get rows affected for player %s (%d): %v", player.Name, player.ID, err)
			return
		}
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

//go:embed asciitball.txt
var chunkyDunker string

func statline(name string) {
	db, err := sql.Open("sqlite3", config.DatabaseFile)
	if err != nil {
		panic(fmt.Errorf("failed to open databse: %v", err))
	}
	defer db.Close()

	var id int
	err = db.QueryRow("SELECT id FROM players WHERE name = $1", name).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		fmt.Println(chunkyDunker)
		fmt.Printf("There is no player with that name. James Naismith wishes %s best of luck in their NBA aspirations.\n", name)
		return
	} else if err != nil {
		panic(err)
	}
	games := nba.LeagueGameFinderByPlayerID(id)
	game := games[0]

	statStrings := []string {
		"Point",
		"Rebound",
		"Assist",
		"Steal",
		"Block",
		"Personal Foul",
		"Turnover",
	}

	stats := []int {
		game.PTS,
		game.REB,
		game.AST,
		game.STL,
		game.BLK,
		game.PF,
		game.TOV,
	}

	statline := []string{}

	if len(stats) != len(statStrings) {
		panic(fmt.Errorf("length of stats (%d) != length of statStrings (%d)", len(stats), len(statStrings)))
	}

	for i := range stats {
		appendAndPluralize(stats[i], statStrings[i], &statline)
	}
	if game.FGA > 0 {
		fg := fmt.Sprintf("%d-%d FG (%s)", game.FGM, game.FGA, floatPercentage(*game.FG_PCT))
		statline = append(statline, fg)
	}
	if game.FG3A > 0 {
		fg3 := fmt.Sprintf("%d-%d 3PT (%s)", game.FG3M, game.FG3A, floatPercentage(*game.FG3_PCT))
		statline = append(statline, fg3)
	}
	if game.FTA > 0 {
		ft := fmt.Sprintf("%d-%d FT (%s)", game.FTM, game.FTA, floatPercentage(*game.FT_PCT))
		statline = append(statline, ft)
	}
	if game.PlusMinus >= 0 {
		pm := fmt.Sprintf("+%d in %d minutes", game.PlusMinus, game.MIN)
		statline = append(statline, pm)
	} else {
		pm := fmt.Sprintf("%d in %d minutes", game.PlusMinus, game.MIN)
		statline = append(statline, pm)
	}

	parsedDate, err := time.Parse("2006-01-02", game.GameDate)
	if err != nil {
		panic(err)
	}
	formatDate := parsedDate.Format("01.02.2006")

	fmt.Printf("%s | %s %s\n", game.PlayerName, game.Matchup, formatDate)
	fmt.Println(strings.Join(statline, ", "))
}

func appendAndPluralize(stat int, statString string, statline *[]string) {
	if stat > 0 {
		s := fmt.Sprintf("%d %s", stat, statString)
		if stat > 1 {
			s += "s"
		}
		*statline = append(*statline, s)
	}
}

func floatPercentage(f float64) string {
	if f*100 == float64(int(f*100)) {
		return fmt.Sprintf("%.f%%", f*100)
	} else if f*1000 == float64(int(f*1000)) {
		return fmt.Sprintf("%.1f%%", f*100)
	} else {
		return fmt.Sprintf("%.2f%%", f*100)
	}
}
