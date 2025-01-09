package main

import (
	"basketball/config"
	"basketball/db"
	"basketball/nba"

	"crypto/md5"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	flag "github.com/spf13/pflag"
)

var statlinePlayerName string
var videoPlayerName string

func init() {
	flag.StringVarP(&statlinePlayerName, "statline", "s", "", "player name to get statline for")
	flag.StringVarP(&videoPlayerName, "video", "v", "", "player name to get video of")
	flag.Parse()
}

func main() {
	config.LoadConfig()
	db.SetupDatabase()
	db.RunMigrations()
	db.ValidateMigrations()

	scrapeCommonAllPlayers()
	if len(videoPlayerName) != 0 {
		video(videoPlayerName)
	}
	if len(statlinePlayerName) != 0 {
		statline(statlinePlayerName)
	}
}

func scrapeCommonAllPlayers() {
	players := nba.CommonAllPlayers()
	db.InsertPlayers(players)
}

func statline(playerCode string) {
	id := db.PlayerIDFromCode(playerCode)
	games := nba.LeagueGameFinderByPlayerID(id)
	game := games[0]
	printStatline(game)
}

func printStatline(game nba.LeagueGameFinderByPlayerGame) {
	statStrings := []string{
		"Point",
		"Rebound",
		"Assist",
		"Steal",
		"Block",
		"Personal Foul",
		"Turnover",
	}
	fmt.Println("GameID:", *game.GameID)
	fmt.Println("PlayerID:", int(*game.PlayerId))
	stats := []float64{
		*game.PTS,
		*game.REB,
		*game.AST,
		*game.STL,
		*game.BLK,
		*game.PF,
		*game.TOV,
	}
	statline := []string{}

	if len(stats) != len(statStrings) {
		panic(fmt.Errorf("length of stats (%d) != length of statStrings (%d)", len(stats), len(statStrings)))
	}

	for i := range stats {
		appendAndPluralize(stats[i], statStrings[i], &statline)
	}
	if game.FGA != nil && *game.FGA > 0 {
		fg := fmt.Sprintf("%d-%d FG (%s)", int(*game.FGM), int(*game.FGA), floatPercentage(*game.FG_PCT))
		statline = append(statline, fg)
	}
	if game.FG3A != nil && *game.FG3A > 0 {
		fg3 := fmt.Sprintf("%d-%d 3PT (%s)", int(*game.FG3M), int(*game.FG3A), floatPercentage(*game.FG3_PCT))
		statline = append(statline, fg3)
	}
	if game.FTA != nil && *game.FTA > 0 {
		ft := fmt.Sprintf("%d-%d FT (%s)", int(*game.FTM), int(*game.FTA), floatPercentage(*game.FT_PCT))
		statline = append(statline, ft)
	}
	if game.PlusMinus != nil && *game.PlusMinus >= 0 {
		pm := fmt.Sprintf("+%d in %d minutes", int(*game.PlusMinus), int(*game.MIN))
		statline = append(statline, pm)
	} else if game.PlusMinus != nil {
		pm := fmt.Sprintf("%d in %d minutes", int(*game.PlusMinus), int(*game.MIN))
		statline = append(statline, pm)
	}

	parsedDate, err := time.Parse("2006-01-02", *game.GameDate)
	if err != nil {
		panic(err)
	}
	formatDate := parsedDate.Format("01.02.2006")

	if *game.PlayerName == "Miles McBride" {
		fmt.Printf("Miles \"Deuce\" McBride | %s %s\n", *game.Matchup, formatDate)
	} else {
		fmt.Printf("%s | %s %s\n", *game.PlayerName, *game.Matchup, formatDate)
	}
	fmt.Println(strings.Join(statline, ", "))
}

func appendAndPluralize(stat float64, statString string, statline *[]string) {
	if stat > 0 {
		s := fmt.Sprintf("%d %s", int(stat), statString)
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

const KnicksTeamId = 1610612752

func video(playerCode string) {
	id := db.PlayerIDFromCode(playerCode)
	games := nba.LeagueGameFinderByPlayerID(id)
	game := games[0]

	measures := []nba.VideoDetailsAssetContextMeasure{
		nba.VideoDetailsAssetContextMeasures.FGA,
		nba.VideoDetailsAssetContextMeasures.REB,
		nba.VideoDetailsAssetContextMeasures.AST,
		nba.VideoDetailsAssetContextMeasures.STL,
		nba.VideoDetailsAssetContextMeasures.TOV,
	}
	gameAssets := map[string][]nba.VideoDetailAsset{}
	errors := []error{}

	for _, m := range measures {
		err := getVideoAssets(game, m, &gameAssets)
		if err != nil {
			errors = append(errors, err)
		}
		time.Sleep(time.Millisecond * 50)
	}

	if len(errors) != 0 {
		fmt.Printf("encountered %d errors when querying for assets\n", len(errors))
	}
	for i, e := range errors {
		var input string
		fmt.Printf("%d/%d:\n", i+1, len(errors))
		fmt.Println(e)
		fmt.Println("Would you like to continue? (y/n)")
		fmt.Scan(&input)
		if !regexp.MustCompile("^[yY]").Match([]byte(input)) {
			os.Exit(1)
		}
	}

	assets := make([]nba.VideoDetailAsset, 0, len(gameAssets))
	for measure := range gameAssets {
		assets = append(assets, gameAssets[measure]...)
	}

	if len(assets) == 0 {
		panic("uh oh no assets found :(")
	}

	sortAssets(&assets)
	tmpDir := mkdirTmp(&game)
	downloadAssets(&assets, tmpDir)

	if err := os.Symlink(config.EndScreenFile, fmt.Sprintf("%s/%06d.mp4", tmpDir, len(assets))); err != nil {
		_ = os.RemoveAll(tmpDir)
		panic(err)
	}

	file, err := os.Create(tmpDir + "/files.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// one extra iteration for end_screen.mp4
	for i := 0; i <= len(assets); i++ {
		file.Write([]byte(fmt.Sprintf("file '%06d.mp4'\n", i)))
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	timeString := fmt.Sprintf("%d", time.Now().Unix())
	sum := md5.Sum([]byte(timeString))
	outputFileName := home + "/Downloads/" + fmt.Sprintf("%x", sum) + ".mp4"

	args := []string{"-f", "concat", "-safe", "0", "-vsync", "0", "-i", fmt.Sprintf("%s/files.txt", tmpDir), "-c", "copy", outputFileName}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdin, cmd.Stderr, cmd.Stdout = os.Stdin, os.Stderr, os.Stdout
	fmt.Println(strings.Join(cmd.Args, " "))

	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		panic(err)
	}

	if err := os.RemoveAll(tmpDir); err != nil {
		panic(err)
	}
	printStatline(game)
}

func getVideoAssets(game nba.LeagueGameFinderByPlayerGame, measure nba.VideoDetailsAssetContextMeasure, gameAssets *map[string][]nba.VideoDetailAsset) error {
	assets := []nba.VideoDetailAsset{}
	for _, a := range nba.VideoDetailsAsset(*game.GameID, *game.PlayerId, *game.TeamID, measure) {
		if a.LargeUrl == nil && a.MedUrl == nil && a.SmallUrl == nil {
			continue
		}
		assets = append(assets, a)
	}

	(*gameAssets)[string(measure)] = assets

	switch measure {
	case "FGA":
		if len(assets) != int(*game.FGA) {
			return fmt.Errorf("expected %d FGA assets, have %d", int(*game.FGA), len(assets))
		}
	case "REB":
		if len(assets) != int(*game.REB) {
			return fmt.Errorf("expected %d REB assets, have %d", int(*game.REB), len(assets))
		}
	case "AST":
		if len(assets) != int(*game.AST) {
			return fmt.Errorf("expected %d AST assets, have %d", int(*game.AST), len(assets))
		}
	case "STL":
		if len(assets) != int(*game.STL) {
			return fmt.Errorf("expected %d STL assets, have %d", int(*game.STL), len(assets))
		}
	case "TOV":
		if len(assets) != int(*game.TOV) {
			return fmt.Errorf("expected %d TOV assets, have %d", int(*game.TOV), len(assets))
		}
	case "PF":
		if len(assets) != int(*game.PF) {
			return fmt.Errorf("expected %d PF assets, have %d", int(*game.PF), len(assets))
		}
	case "FTA":
		if len(assets) != int(*game.FTA) {
			return fmt.Errorf("expected %d FTA assets, have %d", int(*game.FTA), len(assets))
		}
	default:
		return fmt.Errorf("unexpected context measure provided: \"%s\"", string(measure))
	}
	return nil
}

func sortAssets(assets *[]nba.VideoDetailAsset) {
	re := regexp.MustCompile(`(?:https:\/\/videos.nba.com\/nba\/pbp\/media\/\d+\/\d+\/\d+\/)(\d+)\/(\d+)`)
	slices.SortStableFunc(*assets, func(a, b nba.VideoDetailAsset) int {
		var urlA string
		if a.LargeUrl != nil {
			urlA = *a.LargeUrl
		} else if a.MedUrl != nil {
			urlA = *a.MedUrl
		} else if a.SmallUrl != nil {
			urlA = *a.SmallUrl
		} else {
			panic(fmt.Errorf("uh oh this highlight lacks a valid url: %s", *a.Description))
		}

		var urlB string
		if b.LargeUrl != nil {
			urlB = *b.LargeUrl
		} else if b.MedUrl != nil {
			urlB = *b.MedUrl
		} else if b.SmallUrl != nil {
			urlB = *b.SmallUrl
		} else {
			panic(fmt.Errorf("uh oh this highlight lacks a valid url: %s", *b.Description))
		}

		matchesA := re.FindStringSubmatch(urlA)
		matchesB := re.FindStringSubmatch(urlB)

		sortNumA := matchesA[1] + fmt.Sprintf("%03s", matchesA[2])
		sortNumB := matchesB[1] + fmt.Sprintf("%03s", matchesB[2])

		numA, err := strconv.Atoi(sortNumA)
		if err != nil {
			panic(err)
		}
		numB, err := strconv.Atoi(sortNumB)
		if err != nil {
			panic(err)
		}

		return numA - numB
	})
}

func mkdirTmp(game *nba.LeagueGameFinderByPlayerGame) string {
	parsedDate, err := time.Parse("2006-01-02", *(*game).GameDate)
	if err != nil {
		panic(err)
	}
	formatDate := parsedDate.Format("01.02.2006")
	tmpDirPattern := strings.ReplaceAll(*(*game).PlayerName, " ", "_") + "_" + formatDate + "_"
	tmpDir, err := os.MkdirTemp(os.TempDir(), tmpDirPattern)
	if err != nil {
		panic(err)
	}
	return tmpDir
}

func downloadAssets(assets *[]nba.VideoDetailAsset, tmpDir string) {

	fmt.Println("Downloading assets...")
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(*assets))

	for i, asset := range *assets {
		filename := fmt.Sprintf("%s/%06d.mp4", tmpDir, i)
		wg.Add(1)
		go downloadVideoUrl(filename, asset, &wg, errChan)
	}

	wg.Wait()
	close(errChan)

	failure := len(errChan) > 0
	for err := range errChan {
		if err != nil {
			fmt.Println(err)
		}
	}
	if failure {
		_ = os.RemoveAll(tmpDir)
		panic("womp womp")
	}
}

func downloadVideoUrl(filepath string, asset nba.VideoDetailAsset, wg *sync.WaitGroup, errChan chan error) {
	defer wg.Done()

	var url string
	if asset.LargeUrl != nil {
		url = *asset.LargeUrl
	} else if asset.MedUrl != nil {
		url = *asset.MedUrl
	} else {
		url = *asset.SmallUrl
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		errChan <- err
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		errChan <- err
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		errChan <- err
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		errChan <- err
		return
	}
	fmt.Println("Downloaded:", *asset.Description)
}
