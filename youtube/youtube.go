package youtube

import (
	"basketball/config"
	"basketball/utils"
	"encoding/json"

	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func UploadFile(filepath, title, description, playerName, teamName string, service *youtube.Service) {

	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create the video snippet
	snippet := &youtube.VideoSnippet{
		Title:       title,
		Description: description,
		CategoryId:  "17", // 17 => Sports
		Tags:        []string{"basketball", "nba", "NBA", playerName, teamName},
	}

	// Set the privacy status
	status := &youtube.VideoStatus{
		PrivacyStatus:           "public",
		MadeForKids:             false,
		SelfDeclaredMadeForKids: false,
	}

	upload := &youtube.Video{
		Snippet: snippet,
		Status:  status,
	}
	fmt.Println("Uploading to youtube...")
	call := service.Videos.Insert([]string{"snippet", "status"}, upload)
	resp, err := call.Media(file, googleapi.ChunkSize(32*1024*1024)).Do()
	if err != nil {
		panic(err)
	}
	fmt.Println("Upload successful :D!", title, resp.Id)
}

func GetService() (*youtube.Service, error) {
	oauthConfig, err := OAuthConfig()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	token, err := GetToken(oauthConfig)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx := context.Background()
	client := oauthConfig.Client(ctx, token)
	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return service, nil
}

func OAuthConfig() (*oauth2.Config, error) {
	b, err := os.ReadFile(config.SecretFile)
	if err != nil {
		return nil, err
	}

	oauthConfig, err := google.ConfigFromJSON(b, youtube.YoutubeUploadScope)
	if err != nil {
		return nil, err
	}
	return oauthConfig, nil
}

func GetToken(oauthConfig *oauth2.Config) (*oauth2.Token, error) {
	token, err := getTokenFromFile(config.TokenFile)
	if err != nil {
		token, err = getTokenFromWeb(oauthConfig)
		if err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
		SaveToken(config.TokenFile, token)
	} else {
		tokenSource := oauthConfig.TokenSource(context.Background(), token)
		newTok, err := tokenSource.Token()
		if err != nil {
			token, err = getTokenFromWeb(oauthConfig)
			if err != nil {
				return nil, utils.ErrorWithTrace(err)
			}
			SaveToken(config.TokenFile, token)
			return token, nil
		}
		if newTok.AccessToken != token.AccessToken {
			SaveToken(config.TokenFile, newTok)
			token = newTok
		}
	}
	return token, nil
}

func getTokenFromFile(file string) (*oauth2.Token, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	token := &oauth2.Token{}
	err = json.Unmarshal(data, token)
	return token, err
}

func getTokenFromWeb(oauthConfig *oauth2.Config) (*oauth2.Token, error) {
	authURL := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser: \n%v\n", authURL)

	fmt.Printf("Enter authorization code: ")
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %v", err)
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token: %v", err)
	}
	return token, nil
}

func SaveToken(file string, token *oauth2.Token) {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		panic(fmt.Errorf("unable to cache OAuth token: %v", err))
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
