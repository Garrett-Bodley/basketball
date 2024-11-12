package nba

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func init() {
	fmt.Println("The New York Knickerbockers are named after pants")
}

type CommonAllPlayersResp struct {
	ResultSets []struct {
		RowSet [][]interface{} `json:"rowSet"`
	} `json:"resultSets"`
}

type CommonAllPlayer struct {
	ID     int    `json:"PERSON_ID"`
	Name   string `json:"DISPLAY_FIRST_LAST"`
	TeamID int    `json:"TEAM_ID"`
}

func initNBAReq(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Referer", "https://www.nba.com/")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	return req
}

func CommonAllPlayers() []CommonAllPlayer {
	url := "https://stats.nba.com/stats/commonallplayers?LeagueID=00&Season=2023-24&IsOnlyCurrentSeason=0"
	req := initNBAReq(url)

	fmt.Println("Sending CommonAllPlayers request...")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	unmarshalledBody := CommonAllPlayersResp{}
	err = json.Unmarshal(body, &unmarshalledBody)
	if err != nil {
		panic(err)
	}

	players := make([]CommonAllPlayer, len(unmarshalledBody.ResultSets[0].RowSet))
	for i, raw := range unmarshalledBody.ResultSets[0].RowSet {
		player := CommonAllPlayer{}
		player.ID = int(raw[0].(float64))
		player.Name = raw[2].(string)
		player.TeamID = int(raw[8].(float64))
		players[i] = player
	}
	return players
}

type LeagueGameFinderByPlayerIDResp struct {
	ResultsSet []struct {
		Headers []string        `json:"headers"`
		RowSet  [][]interface{} `json:"rowSet"`
	} `json:"resultSets"`
}

type LeagueGameFinderResult struct {
	SeasonID         string
	PlayerId         int
	PlayerName       string
	TeamID           int
	TeamAbbreviation string
	TeamName         string
	GameID           string
	GameDate         string
	Matchup          string
	WL               string
	MIN              int
	PTS              int
	FGM              int
	FGA              int
	FG_PCT           *float64
	FG3M             int
	FG3A             int
	FG3_PCT          *float64
	FTM              int
	FTA              int
	FT_PCT           *float64
	OREB             int
	DREB             int
	REB              int
	AST              int
	STL              int
	BLK              int
	TOV              int
	PF               int
	PlusMinus        int
}

func LeagueGameFinderByPlayerID(playerID int) []LeagueGameFinderResult {
	url := fmt.Sprintf("https://stats.nba.com/stats/leaguegamefinder?PlayerOrTeam=P&PlayerID=%d", playerID)
	req := initNBAReq(url)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	unmarshalledBody := LeagueGameFinderByPlayerIDResp{}
	err = json.Unmarshal(body, &unmarshalledBody)
	if err != nil {
		panic(err)
	}

	expectedHeaders := []string{
		"SEASON_ID",
		"PLAYER_ID",
		"PLAYER_NAME",
		"TEAM_ID",
		"TEAM_ABBREVIATION",
		"TEAM_NAME",
		"GAME_ID",
		"GAME_DATE",
		"MATCHUP",
		"WL",
		"MIN",
		"PTS",
		"FGM",
		"FGA",
		"FG_PCT",
		"FG3M",
		"FG3A",
		"FG3_PCT",
		"FTM",
		"FTA",
		"FT_PCT",
		"OREB",
		"DREB",
		"REB",
		"AST",
		"STL",
		"BLK",
		"TOV",
		"PF",
		"PLUS_MINUS",
	}
	if len(expectedHeaders) != len(unmarshalledBody.ResultsSet[0].Headers) {
		panic(fmt.Errorf("Expected headers to be of length %d, found %d.", len(expectedHeaders), len(unmarshalledBody.ResultsSet[0].Headers)))
	}
	for i := range expectedHeaders {
		if expectedHeaders[i] != unmarshalledBody.ResultsSet[0].Headers[i] {
			panic(fmt.Errorf("Uh Oh! Mismatched headers! Expected %s, found %s", expectedHeaders[i], unmarshalledBody.ResultsSet[0].Headers[i]))
		}
	}

	games := make([]LeagueGameFinderResult, len(unmarshalledBody.ResultsSet[0].RowSet))
	for i, raw := range unmarshalledBody.ResultsSet[0].RowSet {
		game := LeagueGameFinderResult{
			SeasonID:         raw[0].(string),
			PlayerId:         int(raw[1].(float64)),
			PlayerName:       raw[2].(string),
			TeamID:           int(raw[3].(float64)),
			TeamAbbreviation: raw[4].(string),
			TeamName:         raw[5].(string),
			GameID:           raw[6].(string),
			GameDate:         raw[7].(string),
			Matchup:          raw[8].(string),
			WL:               raw[9].(string),
			MIN:              int(raw[10].(float64)),
			PTS:              int(raw[11].(float64)),
			FGM:              int(raw[12].(float64)),
			FGA:              int(raw[13].(float64)),
			FG_PCT:           maybe[float64](raw[14]),
			FG3M:             int(raw[15].(float64)),
			FG3A:             int(raw[16].(float64)),
			FG3_PCT:          maybe[float64](raw[17]),
			FTM:              int(raw[18].(float64)),
			FTA:              int(raw[19].(float64)),
			FT_PCT:           maybe[float64](raw[20]),
			OREB:             int(raw[21].(float64)),
			DREB:             int(raw[22].(float64)),
			REB:              int(raw[23].(float64)),
			AST:              int(raw[24].(float64)),
			STL:              int(raw[25].(float64)),
			BLK:              int(raw[26].(float64)),
			TOV:              int(raw[27].(float64)),
			PF:               int(raw[28].(float64)),
			PlusMinus:        int(raw[29].(float64)),
		}
		games[i] = game
	}
	return games
}

func maybe[T any](x any) *T {
	if x, ok := x.(T); ok {
		return &x
	}
	return nil
}
