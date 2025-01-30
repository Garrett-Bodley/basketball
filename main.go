package main

import (
	"basketball/config"
	"basketball/db"
	"basketball/nba"
	"basketball/youtube"

	"crypto/md5"
	_ "embed"
	"fmt"
	"io"
	"math"
	"math/rand"
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
var Knicks *bool

func init() {
	flag.StringVarP(&statlinePlayerName, "statline", "s", "", "player name to get statline for")
	flag.StringVarP(&videoPlayerName, "video", "v", "", "player name to get video of")
	Knicks = flag.BoolP("knicks", "k", false, "downloads and uploads all knicks highlights")
	flag.Parse()
}

func main() {
	config.LoadConfig()
	db.SetupDatabase()
	db.RunMigrations()
	db.ValidateMigrations()

	// scrapeCommonAllPlayers()
	if *Knicks {
		Knickerbockers()
	}
	if len(videoPlayerName) != 0 {
		videoRes, err := Video(videoPlayerName, true)
		if err != nil {
			panic(err)
		}
		printStatline(videoRes.Game)
	}
	if len(statlinePlayerName) != 0 {
		if err := Statline(statlinePlayerName); err != nil {
			panic(err)
		}
	}
}

const KnicksTeamId = 1610612752

func Knickerbockers() {
	games := nba.LeagueGameFinderByTeamID(KnicksTeamId)
	game := games[0]
	boxscore := nba.BoxScoreTraditionalV2(*game.GameID)

	wg := sync.WaitGroup{}
	errMap := sync.Map{}
	videoMap := sync.Map{}

	fmt.Println("Fetching resources")
	for _, p := range boxscore.PlayerStats {
		if int(*p.TeamId) != KnicksTeamId {
			continue
		}
		if p.MIN == nil {
			continue
		}
		split := strings.Split(*p.MIN, ":")
		minStr, secStr := split[0], split[1]
		minFloat, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			panic(err)
		}
		secFloat, err := strconv.ParseFloat(secStr, 64)
		if err != nil {
			panic(err)
		}
		min, sec := int(minFloat), int(secFloat)
		if min > 0 || sec > 0 {
			wg.Add(1)
			go func() {
				defer func() { wg.Done() }()
				res, err := Video(*p.PlayerName, false)
				if err != nil {
					errMap.Store(*p.PlayerName, err)
				} else {
					videoMap.Store(*p.PlayerName, res)
				}
			}()
		}
	}
	wg.Wait()

	fmt.Println("Displaying errors...")
	errMap.Range(func(key, value any) bool {
		// errFlag = true
		fmt.Println("player error:", key.(string))
		fmt.Println(value.(error))
		return true
	})

	// if errFlag{
	var input string
	fmt.Println("Upload to Youtube? (y/n)")
	fmt.Scanln(&input)
	if !regexp.MustCompile("^[yY]").Match([]byte(input)) {
		videoMap.Range(func(key, value any) bool {
			videoRes := value.(VideoRes)
			_ = os.Remove(videoRes.OutputFile)
			return true
		})
		return
	}
	// }
	oauthConfig, err := youtube.OAuthConfig()
	if err != nil {
		videoMap.Range(func(key, value any) bool {
			videoRes := value.(VideoRes)
			_ = os.Remove(videoRes.OutputFile)
			return true
		})
		panic(err)
	}
	token, err := youtube.GetToken(oauthConfig)
	if err != nil {
		videoMap.Range(func(key, value any) bool {
			videoRes := value.(VideoRes)
			_ = os.Remove(videoRes.OutputFile)
			return true
		})
		panic(err)
	}

	videoMap.Range(func(playerName, value any) bool {
		videoRes := value.(VideoRes)
		game := videoRes.Game
		title, err := title(game)
		if err != nil {
			fmt.Println("failed while trying to generate youtube video title for", playerName.(string))
			fmt.Println(err)
			return true
		}
		description, err := statString(game)
		if err != nil {
			fmt.Println("failed while trying to generate youtube video description for", playerName.(string))
			fmt.Println(err)
			return true
		}

		wg.Add(1)
		go func() {
			defer func() { wg.Done() }()
			youtube.UploadFile(videoRes.OutputFile, title, description, *game.PlayerName, *game.TeamName, oauthConfig, token)
			_ = os.Remove(videoRes.OutputFile)
		}()
		return true
	})
	wg.Wait()
}

func scrapeCommonAllPlayers() {
	players := nba.CommonAllPlayers()
	db.InsertPlayers(players)
}

func Statline(playerCode string) error {
	id, err := db.PlayerIDFromCode(playerCode)
	if err != nil {
		return err
	}
	games, err := nba.LeagueGameFinderByPlayerID(id)
	if err != nil {
		return err
	}
	game := games[0]
	printStatline(game)
	return nil
}

func printStatline(game nba.LeagueGameFinderGame) {
	fmt.Println("GameID:", *game.GameID)
	fmt.Println("PlayerID:", int(*game.PlayerId))
	title, err := title(game)
	if err != nil {
		panic(err)
	}
	statline, err := statString(game)
	if err != nil {
		panic(err)
	}
	fmt.Println(title)
	fmt.Println(statline)
}

func statString(game nba.LeagueGameFinderGame) (string, error) {
	statStrings := []string{
		"Point",
		"Rebound",
		"Assist",
		"Steal",
		"Block",
		"Personal Foul",
		"Turnover",
	}
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
		return "", fmt.Errorf("length of stats (%d) != length of statStrings (%d)", len(stats), len(statStrings))
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
	return strings.Join(statline, ", "), nil
}

func title(game nba.LeagueGameFinderGame) (string, error) {
	parsedDate, err := time.Parse("2006-01-02", *game.GameDate)
	if err != nil {
		return "", err
	}

	formatDate := parsedDate.Format("01.02.2006")

	if *game.PlayerName == "Miles McBride" {
		return fmt.Sprintf("Miles \"Deuce\" McBride | %s %s", *game.Matchup, formatDate), nil
	} else {
		return fmt.Sprintf("%s | %s %s", *game.PlayerName, *game.Matchup, formatDate), nil
	}
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

type VideoRes struct {
	Game       nba.LeagueGameFinderGame
	OutputFile string
}

func Video(playerCode string, toDownloadsDir bool) (VideoRes, error) {
	id, err := db.PlayerIDFromCode(playerCode)
	res := VideoRes{}
	if err != nil {
		return res, err
	}
	games, err := nba.LeagueGameFinderByPlayerID(id)
	if err != nil {
		return res, err
	}
	res.Game = games[0]

	measures := []nba.VideoDetailsAssetContextMeasure{
		nba.VideoDetailsAssetContextMeasures.FGA,
		nba.VideoDetailsAssetContextMeasures.REB,
		nba.VideoDetailsAssetContextMeasures.AST,
		nba.VideoDetailsAssetContextMeasures.STL,
		nba.VideoDetailsAssetContextMeasures.TOV,
		nba.VideoDetailsAssetContextMeasures.BLK,
	}
	assets, err := getVideoAssets(res.Game, measures)
	if err != nil {
		return res, err
	}
	if err := sortAssets(&assets); err != nil {
		return res, err
	}
	tmpDir, err := mkdirTmp(&res.Game)
	if err != nil {
		return res, err
	}
	if err := downloadAssets(&assets, tmpDir); err != nil {
		return res, err
	}
	outputFile, err := ffmpeg(tmpDir, len(assets))
	if err != nil {
		return res, err
	}
	if toDownloadsDir {
		timeString := fmt.Sprintf("%d", time.Now().Unix())
		sum := md5.Sum([]byte(timeString))
		home, err := os.UserHomeDir()
		if err != nil {
			return res, err
		}
		downloadFile := home + "/Downloads/" + fmt.Sprintf("%x.mp4", sum)
		os.Rename(outputFile, downloadFile)
		outputFile = downloadFile
	}
	res.OutputFile = outputFile
	return res, nil
}

var getVideoAssetsSem = make(chan int, 1)

func getVideoAssets(game nba.LeagueGameFinderGame, measures []nba.VideoDetailsAssetContextMeasure) ([]nba.VideoDetailAsset, error) {
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(measures))
	gaMu := sync.Mutex{}
	gameAssets := []nba.VideoDetailAsset{}
	for _, m := range measures {
		wg.Add(1)
		go func() {
			defer wg.Done()
			measureAssets, err := getVideoAssetsByMeasure(game, m)
			if err != nil {
				errChan <- err
			}
			gaMu.Lock()
			gameAssets = append(gameAssets, measureAssets...)
			gaMu.Unlock()
		}()
	}

	wg.Wait()
	close(errChan)

	getVideoAssetsSem <- 1
	defer func() { <-getVideoAssetsSem }()

	n := len(errChan)
	if n != 0 {
		fmt.Println(*game.PlayerName)
		fmt.Printf("encountered %d errors when querying for assets\n", len(errChan))
	}
	i := 0
	for e := range errChan {
		var input string
		fmt.Printf("%d/%d:\n", i+1, n)
		fmt.Println(e)
		fmt.Println("Would you like to continue? (y/n)")
		fmt.Scan(&input)
		if !regexp.MustCompile("^[yY]").Match([]byte(input)) {
			return []nba.VideoDetailAsset{}, fmt.Errorf("user aborted: %s", e.Error())
		}
		i++
	}

	if len(gameAssets) == 0 {
		return nil, fmt.Errorf("no assets found")
	}

	return gameAssets, nil
}

func getVideoAssetsByMeasure(game nba.LeagueGameFinderGame, measure nba.VideoDetailsAssetContextMeasure) ([]nba.VideoDetailAsset, error) {
	measureAssets := []nba.VideoDetailAsset{}
	apiRes, err := nba.VideoDetailsAsset(*game.GameID, *game.PlayerId, *game.TeamID, measure)
	if err != nil {
		return measureAssets, err
	}

	// filter out assets with no URL
	for _, a := range apiRes {
		if a.LargeUrl == nil && a.MedUrl == nil && a.SmallUrl == nil {
			continue
		}
		measureAssets = append(measureAssets, a)
	}

	switch measure {
	case "FGA":
		if len(measureAssets) != int(*game.FGA) {
			return measureAssets, fmt.Errorf("expected %d FGA assets, have %d", int(*game.FGA), len(measureAssets))
		}
	case "REB":
		if len(measureAssets) != int(*game.REB) {
			return measureAssets, fmt.Errorf("expected %d REB assets, have %d", int(*game.REB), len(measureAssets))
		}
	case "AST":
		if len(measureAssets) != int(*game.AST) {
			return measureAssets, fmt.Errorf("expected %d AST assets, have %d", int(*game.AST), len(measureAssets))
		}
	case "STL":
		if len(measureAssets) != int(*game.STL) {
			return measureAssets, fmt.Errorf("expected %d STL assets, have %d", int(*game.STL), len(measureAssets))
		}
	case "TOV":
		if len(measureAssets) != int(*game.TOV) {
			return measureAssets, fmt.Errorf("expected %d TOV assets, have %d", int(*game.TOV), len(measureAssets))
		}
	case "PF":
		if len(measureAssets) != int(*game.PF) {
			return measureAssets, fmt.Errorf("expected %d PF assets, have %d", int(*game.PF), len(measureAssets))
		}
	case "FTA":
		if len(measureAssets) != int(*game.FTA) {
			return measureAssets, fmt.Errorf("expected %d FTA assets, have %d", int(*game.FTA), len(measureAssets))
		}
	case "BLK":
		if len(measureAssets) != int(*game.BLK) {
			return measureAssets, fmt.Errorf("expected %d BLK assets, have %d", int(*game.BLK), len(measureAssets))
		}
	default:
		return measureAssets, fmt.Errorf("unexpected context measure provided: \"%s\"", string(measure))
	}
	return measureAssets, nil
}

func sortAssets(assets *[]nba.VideoDetailAsset) error {
	re := regexp.MustCompile(`(?:https:\/\/videos.nba.com\/nba\/pbp\/media\/\d+\/\d+\/\d+\/)(\d+)\/(\d+)`)
	errors := []error{}
	slices.SortStableFunc(*assets, func(a, b nba.VideoDetailAsset) int {
		var urlA string
		if a.LargeUrl != nil {
			urlA = *a.LargeUrl
		} else if a.MedUrl != nil {
			urlA = *a.MedUrl
		} else if a.SmallUrl != nil {
			urlA = *a.SmallUrl
		} else {
			errors = append(errors, fmt.Errorf("uh oh this highlight lacks a valid url: %s", *a.Description))
			return 0
		}

		var urlB string
		if b.LargeUrl != nil {
			urlB = *b.LargeUrl
		} else if b.MedUrl != nil {
			urlB = *b.MedUrl
		} else if b.SmallUrl != nil {
			urlB = *b.SmallUrl
		} else {
			errors = append(errors, fmt.Errorf("uh oh this highlight lacks a valid url: %s", *b.Description))
			return 0
		}

		matchesA := re.FindStringSubmatch(urlA)
		matchesB := re.FindStringSubmatch(urlB)

		sortNumA := matchesA[1] + fmt.Sprintf("%03s", matchesA[2])
		sortNumB := matchesB[1] + fmt.Sprintf("%03s", matchesB[2])

		numA, err := strconv.Atoi(sortNumA)
		if err != nil {
			errors = append(errors, err)
			return 0
		}
		numB, err := strconv.Atoi(sortNumB)
		if err != nil {
			errors = append(errors, err)
			return 0
		}

		return numA - numB
	})
	if len(errors) > 0 {
		return gigaError(errors)
	}
	return nil
}

func mkdirTmp(game *nba.LeagueGameFinderGame) (string, error) {
	parsedDate, err := time.Parse("2006-01-02", *(*game).GameDate)
	if err != nil {
		return "", err
	}
	formatDate := parsedDate.Format("01.02.2006")
	tmpDirPattern := strings.ReplaceAll(*(*game).PlayerName, " ", "_") + "_" + formatDate + "_"
	tmpDir, err := os.MkdirTemp(os.TempDir(), tmpDirPattern)
	if err != nil {
		return "", err
	}
	return tmpDir, nil
}

func downloadAssets(assets *[]nba.VideoDetailAsset, tmpDir string) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(*assets))

	for i, asset := range *assets {
		filename := fmt.Sprintf("%s/%06d.mp4", tmpDir, i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := downloadVideoUrl(filename, asset)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	errors := []error{}
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		_ = os.RemoveAll(tmpDir)
		return gigaError(errors)
	}
	return nil
}

func downloadVideoUrl(filepath string, asset nba.VideoDetailAsset) error {
	var url string
	if asset.LargeUrl != nil {
		url = *asset.LargeUrl
	} else if asset.MedUrl != nil {
		url = *asset.MedUrl
	} else {
		url = *asset.SmallUrl
	}
	if err := curlToFile(url, filepath); err != nil {
		return err
	}
	return nil
}

// ffmpeg is written in c and assembly language
func ffmpeg(tmpDir string, count int) (string, error) {
	if err := os.Symlink(config.EndScreenFile, fmt.Sprintf("%s/%06d.mp4", tmpDir, count)); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", err
	}

	file, err := os.Create(tmpDir + "/files.txt")
	if err != nil {
		return "", err
	}
	defer file.Close()

	for i := 0; i <= count; i++ {
		_, err := file.Write([]byte(fmt.Sprintf("file '%06d.mp4'\n", i)))
		if err != nil {
			return "", err
		}
	}

	timeString := fmt.Sprintf("%d%d", time.Now().Unix(), rand.Intn(math.MaxInt64))
	sum := md5.Sum([]byte(timeString))
	outputFileName := os.TempDir() + fmt.Sprintf("%x", sum) + ".mp4"

	args := []string{"-hide_banner", "-v", "fatal", "-f", "concat", "-safe", "0", "-vsync", "0", "-i", fmt.Sprintf("%s/files.txt", tmpDir), "-c", "copy", outputFileName}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdin, cmd.Stderr, cmd.Stdout = os.Stdin, os.Stderr, os.Stdout
	// fmt.Println(strings.Join(cmd.Args, " "))

	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		_ = os.Remove(outputFileName)
		return "", err
	}
	_ = os.RemoveAll(tmpDir)
	return outputFileName, nil
}

func gigaError(slice []error) error {
	errBytes := []byte{}
	for i := range slice {
		errBytes = append(errBytes, []byte(slice[i].Error())...)
		errBytes = append(errBytes, '\n')
	}
	return fmt.Errorf("%s", string(errBytes))
}

var sem = make(chan int, 50)

func curlToFile(url, filepath string) error {
	sem <- 1
	defer func() { <-sem }()
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
