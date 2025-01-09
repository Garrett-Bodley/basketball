package nba

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	PersonID                *float64
	DisplayLastFirst        *string
	DisplayFirstLast        *string
	RosterStatus            *float64
	FromYear                *string
	ToYear                  *string
	PlayerCode              *string
	PlayerSlug              *string
	TeamID                  *float64
	TeamCity                *string
	TeamName                *string
	TeamAbbreviation        *string
	TeamCode                *string
	TeamSlug                *string
	GamesPlayedFlag         *string
	OtherLeagueExperienceCh *string
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
		player := CommonAllPlayer{
			PersonID:                maybe[float64](raw[0]),
			DisplayLastFirst:        maybe[string](raw[1]),
			DisplayFirstLast:        maybe[string](raw[2]),
			RosterStatus:            maybe[float64](raw[3]),
			FromYear:                maybe[string](raw[4]),
			ToYear:                  maybe[string](raw[5]),
			PlayerCode:              maybe[string](raw[6]),
			PlayerSlug:              maybe[string](raw[7]),
			TeamID:                  maybe[float64](raw[8]),
			TeamCity:                maybe[string](raw[9]),
			TeamName:                maybe[string](raw[10]),
			TeamAbbreviation:        maybe[string](raw[11]),
			TeamCode:                maybe[string](raw[12]),
			TeamSlug:                maybe[string](raw[13]),
			GamesPlayedFlag:         maybe[string](raw[14]),
			OtherLeagueExperienceCh: maybe[string](raw[15]),
		}
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

type LeagueGameFinderByPlayerGame struct {
	SeasonID         *string
	PlayerId         *float64
	PlayerName       *string
	TeamID           *float64
	TeamAbbreviation *string
	TeamName         *string
	GameID           *string
	GameDate         *string
	Matchup          *string
	WL               *string
	MIN              *float64
	PTS              *float64
	FGM              *float64
	FGA              *float64
	FG_PCT           *float64
	FG3M             *float64
	FG3A             *float64
	FG3_PCT          *float64
	FTM              *float64
	FTA              *float64
	FT_PCT           *float64
	OREB             *float64
	DREB             *float64
	REB              *float64
	AST              *float64
	STL              *float64
	BLK              *float64
	TOV              *float64
	PF               *float64
	PlusMinus        *float64
}

// gameID: 0022400014
// jalen brunson ID: 1628973
// knicks teamID: 1610612752

func LeagueGameFinderByPlayerID(playerID int) []LeagueGameFinderByPlayerGame {
	url := fmt.Sprintf("https://stats.nba.com/stats/leaguegamefinder?PlayerOrTeam=P&PlayerID=%d", playerID)
	req := initNBAReq(url)
	body := curl(req)

	unmarshalledBody := LeagueGameFinderByPlayerIDResp{}
	err := json.Unmarshal(body, &unmarshalledBody)
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
		panic(fmt.Errorf("expected headers to be of length %d, found %d", len(expectedHeaders), len(unmarshalledBody.ResultsSet[0].Headers)))
	}
	for i := range expectedHeaders {
		if expectedHeaders[i] != unmarshalledBody.ResultsSet[0].Headers[i] {
			panic(fmt.Errorf("uh oh! mismatched headers! expected %s, found %s", expectedHeaders[i], unmarshalledBody.ResultsSet[0].Headers[i]))
		}
	}

	games := make([]LeagueGameFinderByPlayerGame, len(unmarshalledBody.ResultsSet[0].RowSet))
	for i, raw := range unmarshalledBody.ResultsSet[0].RowSet {
		game := LeagueGameFinderByPlayerGame{
			SeasonID:         maybe[string](raw[0]),
			PlayerId:         maybe[float64](raw[1]),
			PlayerName:       maybe[string](raw[2]),
			TeamID:           maybe[float64](raw[3]),
			TeamAbbreviation: maybe[string](raw[4]),
			TeamName:         maybe[string](raw[5]),
			GameID:           maybe[string](raw[6]),
			GameDate:         maybe[string](raw[7]),
			Matchup:          maybe[string](raw[8]),
			WL:               maybe[string](raw[9]),
			MIN:              maybe[float64](raw[10]),
			PTS:              maybe[float64](raw[11]),
			FGM:              maybe[float64](raw[12]),
			FGA:              maybe[float64](raw[13]),
			FG_PCT:           maybe[float64](raw[14]),
			FG3M:             maybe[float64](raw[15]),
			FG3A:             maybe[float64](raw[16]),
			FG3_PCT:          maybe[float64](raw[17]),
			FTM:              maybe[float64](raw[18]),
			FTA:              maybe[float64](raw[19]),
			FT_PCT:           maybe[float64](raw[20]),
			OREB:             maybe[float64](raw[21]),
			DREB:             maybe[float64](raw[22]),
			REB:              maybe[float64](raw[23]),
			AST:              maybe[float64](raw[24]),
			STL:              maybe[float64](raw[25]),
			BLK:              maybe[float64](raw[26]),
			TOV:              maybe[float64](raw[27]),
			PF:               maybe[float64](raw[28]),
			PlusMinus:        maybe[float64](raw[29]),
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

type VideoDetailsAssetResp struct {
	ResultSets struct {
		Meta struct {
			VideoUrls []VideoDetailsAssetURLEntry `json:"videoUrls"`
		} `json:"Meta"`
		Playlist []VideoDetailsAssetPlaylistEntry `json:"playlist"`
	} `json:"resultSets"`
}

type VideoDetailsAssetURLEntry struct {
	Uuid           *string  `json:"uuid"`
	SmallDur       *float64 `json:"sdur"`
	SmallUrl       *string  `json:"surl"`
	SmallThumbnail *string  `json:"sth"`
	MedDur         *float64 `json:"mdur"`
	MedUrl         *string  `json:"murl"`
	MedThumbnail   *string  `json:"mth"`
	LargeDur       *float64 `json:"ldur"`
	LargeUrl       *string  `json:"lurl"`
	LargeThumbnail *string  `json:"lth"`
	Vtt            *string  `json:"vtt"`
	Scc            *string  `json:"scc"`
	Srt            *string  `json:"srt"`
}

type VideoDetailsAssetPlaylistEntry struct {
	GameID               *string  `json:"gi"`
	EventID              *float64 `json:"ei"`
	Year                 *float64 `json:"y"`
	Month                *string  `json:"m"`
	Day                  *string  `json:"d"`
	GameCode             *string  `json:"gc"`
	Period               *float64 `json:"p"`
	Description          *string  `json:"dsc"`
	HomeAbbreviation     *string  `json:"ha"`
	HomeID               *float64 `json:"hid"`
	VisitingAbbreviation *string  `json:"va"`
	VisitingID           *float64 `json:"vid"`
	HomePointsBefore     *float64 `json:"hpb"`
	HomePointsAfter      *float64 `json:"hpa"`
	VisitingPointsBefore *float64 `json:"vpb"`
	VisitingPointsAfter  *float64 `json:"vpa"`
	IdkWhatThisDoes      *float64 `json:"pta"`
}

type VideoDetailAsset struct {
	GameID      *string
	EventID     *float64
	Year        *float64
	Month       *string
	Day         *string
	Description *string
	Uuid        *string
	LargeUrl    *string
	MedUrl      *string
	SmallUrl    *string
}

type VideoDetailsAssetContextMeasure string

var VideoDetailsAssetContextMeasures = struct {
	FGM                VideoDetailsAssetContextMeasure
	FGA                VideoDetailsAssetContextMeasure
	FG_PCT             VideoDetailsAssetContextMeasure
	FG3M               VideoDetailsAssetContextMeasure
	FG3A               VideoDetailsAssetContextMeasure
	FG3_PCT            VideoDetailsAssetContextMeasure
	FTM                VideoDetailsAssetContextMeasure
	FTA                VideoDetailsAssetContextMeasure
	OREB               VideoDetailsAssetContextMeasure
	DREB               VideoDetailsAssetContextMeasure
	AST                VideoDetailsAssetContextMeasure
	FGM_AST            VideoDetailsAssetContextMeasure
	FG3_AST            VideoDetailsAssetContextMeasure
	STL                VideoDetailsAssetContextMeasure
	BLK                VideoDetailsAssetContextMeasure
	BLKA               VideoDetailsAssetContextMeasure
	TOV                VideoDetailsAssetContextMeasure
	PF                 VideoDetailsAssetContextMeasure
	PFD                VideoDetailsAssetContextMeasure
	POSS_END_FT        VideoDetailsAssetContextMeasure
	PTS_PAINT          VideoDetailsAssetContextMeasure
	PTS_FB             VideoDetailsAssetContextMeasure
	PTS_OFF_TOV        VideoDetailsAssetContextMeasure
	PTS_2ND_CHANCE     VideoDetailsAssetContextMeasure
	REB                VideoDetailsAssetContextMeasure
	TM_FGM             VideoDetailsAssetContextMeasure
	TM_FGA             VideoDetailsAssetContextMeasure
	TM_FG3M            VideoDetailsAssetContextMeasure
	TM_FG3A            VideoDetailsAssetContextMeasure
	TM_FTM             VideoDetailsAssetContextMeasure
	TM_FTA             VideoDetailsAssetContextMeasure
	TM_OREB            VideoDetailsAssetContextMeasure
	TM_DREB            VideoDetailsAssetContextMeasure
	TM_REB             VideoDetailsAssetContextMeasure
	TM_TEAM_REB        VideoDetailsAssetContextMeasure
	TM_AST             VideoDetailsAssetContextMeasure
	TM_STL             VideoDetailsAssetContextMeasure
	TM_BLK             VideoDetailsAssetContextMeasure
	TM_BLKA            VideoDetailsAssetContextMeasure
	TM_TOV             VideoDetailsAssetContextMeasure
	TM_TEAM_TOV        VideoDetailsAssetContextMeasure
	TM_PF              VideoDetailsAssetContextMeasure
	TM_PFD             VideoDetailsAssetContextMeasure
	TM_PTS             VideoDetailsAssetContextMeasure
	TM_PTS_PAINT       VideoDetailsAssetContextMeasure
	TM_PTS_FB          VideoDetailsAssetContextMeasure
	TM_PTS_OFF_TOV     VideoDetailsAssetContextMeasure
	TM_PTS_2ND_CHANCE  VideoDetailsAssetContextMeasure
	TM_FGM_AST         VideoDetailsAssetContextMeasure
	TM_FG3_AST         VideoDetailsAssetContextMeasure
	TM_POSS_END_FT     VideoDetailsAssetContextMeasure
	OPP_FGM            VideoDetailsAssetContextMeasure
	OPP_FGA            VideoDetailsAssetContextMeasure
	OPP_FG3M           VideoDetailsAssetContextMeasure
	OPP_FG3A           VideoDetailsAssetContextMeasure
	OPP_FTM            VideoDetailsAssetContextMeasure
	OPP_FTA            VideoDetailsAssetContextMeasure
	OPP_OREB           VideoDetailsAssetContextMeasure
	OPP_DREB           VideoDetailsAssetContextMeasure
	OPP_REB            VideoDetailsAssetContextMeasure
	OPP_TEAM_REB       VideoDetailsAssetContextMeasure
	OPP_AST            VideoDetailsAssetContextMeasure
	OPP_STL            VideoDetailsAssetContextMeasure
	OPP_BLK            VideoDetailsAssetContextMeasure
	OPP_BLKA           VideoDetailsAssetContextMeasure
	OPP_TOV            VideoDetailsAssetContextMeasure
	OPP_TEAM_TOV       VideoDetailsAssetContextMeasure
	OPP_PF             VideoDetailsAssetContextMeasure
	OPP_PFD            VideoDetailsAssetContextMeasure
	OPP_PTS            VideoDetailsAssetContextMeasure
	OPP_PTS_PAINT      VideoDetailsAssetContextMeasure
	OPP_PTS_FB         VideoDetailsAssetContextMeasure
	OPP_PTS_OFF_TOV    VideoDetailsAssetContextMeasure
	OPP_PTS_2ND_CHANCE VideoDetailsAssetContextMeasure
	OPP_FGM_AST        VideoDetailsAssetContextMeasure
	OPP_FG3_AST        VideoDetailsAssetContextMeasure
	OPP_POSS_END_FT    VideoDetailsAssetContextMeasure
	PTS                VideoDetailsAssetContextMeasure
}{
	FGM:                "FGM",
	FGA:                "FGA",
	FG_PCT:             "FG_PCT",
	FG3M:               "FG3M",
	FG3A:               "FG3A",
	FG3_PCT:            "FG3_PCT",
	FTM:                "FTM",
	FTA:                "FTA",
	OREB:               "OREB",
	DREB:               "DREB",
	AST:                "AST",
	FGM_AST:            "FGM_AST",
	FG3_AST:            "FG3_AST",
	STL:                "STL",
	BLK:                "BLK",
	BLKA:               "BLKA",
	TOV:                "TOV",
	PF:                 "PF",
	PFD:                "PFD",
	POSS_END_FT:        "POSS_END_FT",
	PTS_PAINT:          "PTS_PAINT",
	PTS_FB:             "PTS_FB",
	PTS_OFF_TOV:        "PTS_OFF_TOV",
	PTS_2ND_CHANCE:     "PTS_2ND_CHANCE",
	REB:                "REB",
	TM_FGM:             "TM_FGM",
	TM_FGA:             "TM_FGA",
	TM_FG3M:            "TM_FG3M",
	TM_FG3A:            "TM_FG3A",
	TM_FTM:             "TM_FTM",
	TM_FTA:             "TM_FTA",
	TM_OREB:            "TM_OREB",
	TM_DREB:            "TM_DREB",
	TM_REB:             "TM_REB",
	TM_TEAM_REB:        "TM_TEAM_REB",
	TM_AST:             "TM_AST",
	TM_STL:             "TM_STL",
	TM_BLK:             "TM_BLK",
	TM_BLKA:            "TM_BLKA",
	TM_TOV:             "TM_TOV",
	TM_TEAM_TOV:        "TM_TEAM_TOV",
	TM_PF:              "TM_PF",
	TM_PFD:             "TM_PFD",
	TM_PTS:             "TM_PTS",
	TM_PTS_PAINT:       "TM_PTS_PAINT",
	TM_PTS_FB:          "TM_PTS_FB",
	TM_PTS_OFF_TOV:     "TM_PTS_OFF_TOV",
	TM_PTS_2ND_CHANCE:  "TM_PTS_2ND_CHANCE",
	TM_FGM_AST:         "TM_FGM_AST",
	TM_FG3_AST:         "TM_FG3_AST",
	TM_POSS_END_FT:     "TM_POSS_END_FT",
	OPP_FGM:            "OPP_FGM",
	OPP_FGA:            "OPP_FGA",
	OPP_FG3M:           "OPP_FG3M",
	OPP_FG3A:           "OPP_FG3A",
	OPP_FTM:            "OPP_FTM",
	OPP_FTA:            "OPP_FTA",
	OPP_OREB:           "OPP_OREB",
	OPP_DREB:           "OPP_DREB",
	OPP_REB:            "OPP_REB",
	OPP_TEAM_REB:       "OPP_TEAM_REB",
	OPP_AST:            "OPP_AST",
	OPP_STL:            "OPP_STL",
	OPP_BLK:            "OPP_BLK",
	OPP_BLKA:           "OPP_BLKA",
	OPP_TOV:            "OPP_TOV",
	OPP_TEAM_TOV:       "OPP_TEAM_TOV",
	OPP_PF:             "OPP_PF",
	OPP_PFD:            "OPP_PFD",
	OPP_PTS:            "OPP_PTS",
	OPP_PTS_PAINT:      "OPP_PTS_PAINT",
	OPP_PTS_FB:         "OPP_PTS_FB",
	OPP_PTS_OFF_TOV:    "OPP_PTS_OFF_TOV",
	OPP_PTS_2ND_CHANCE: "OPP_PTS_2ND_CHANCE",
	OPP_FGM_AST:        "OPP_FGM_AST",
	OPP_FG3_AST:        "OPP_FG3_AST",
	OPP_POSS_END_FT:    "OPP_POSS_END_FT",
	PTS:                "PTS",
}

func VideoDetailsAsset(gameID string, playerID, teamID float64, contextMeasure VideoDetailsAssetContextMeasure) []VideoDetailAsset {
	url := fmt.Sprintf("https://stats.nba.com/stats/videodetailsasset?AheadBehind=&ClutchTime=&ContextFilter=&ContextMeasure=%s&DateFrom=&DateTo=&EndPeriod=&EndRange=&GameID=%s&GameSegment=&LastNGames=0&LeagueID=&Location=&Month=0&OpponentTeamID=0&Outcome=&Period=0&PlayerID=%d&PointDiff=&Position=&RangeType=&RookieYear=&Season=2024-25&SeasonSegment=&SeasonType=Regular+Season&StartPeriod=&StartRange=&TeamID=%d&VsConference=&VsDivision=", contextMeasure, gameID, int(playerID), int(teamID))
	req := initNBAReq(url)
	body := curl(req)

	unmarshalledBody := VideoDetailsAssetResp{}
	err := json.Unmarshal(body, &unmarshalledBody)
	if err != nil && strings.Contains(err.Error(), "invalid character '<'") {
		fmt.Println(string(body))
		panic(err)
	} else if err != nil {
		panic(err)
	}

	Playlist := unmarshalledBody.ResultSets.Playlist
	VideoUrls := unmarshalledBody.ResultSets.Meta.VideoUrls

	if len(Playlist) != len(VideoUrls) {
		panic("playlist array and urls array lengths do not match (╯°□°)╯︵ ɹoɹɹƎ")
	}

	res := make([]VideoDetailAsset, 0, len(Playlist))
	for i := range Playlist {
		entry := VideoDetailAsset{
			GameID:      Playlist[i].GameID,
			EventID:     Playlist[i].EventID,
			Year:        Playlist[i].Year,
			Month:       Playlist[i].Month,
			Day:         Playlist[i].Day,
			Description: Playlist[i].Description,
			Uuid:        VideoUrls[i].Uuid,
			SmallUrl:    VideoUrls[i].SmallUrl,
			MedUrl:      VideoUrls[i].MedUrl,
			LargeUrl:    VideoUrls[i].LargeUrl,
		}
		if entry.LargeUrl == nil && entry.MedUrl == nil && entry.SmallUrl == nil {
			continue
		}
		res = append(res, entry)
	}
	return res
}

func curl(req *http.Request) []byte {
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
	return body
}
