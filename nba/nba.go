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

func CommonAllPlayers() []CommonAllPlayer {
	url := "https://stats.nba.com/stats/commonallplayers?LeagueID=00&Season=2023-24&IsOnlyCurrentSeason=0"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Referer", "https://www.nba.com/")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	fmt.Println("Sending request...")
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
