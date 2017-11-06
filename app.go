package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

const atcoder_url = "https://atcoder.jp/?lang=ja"
const web_hook = "hoge"

func tokenCacheFile() string {
	tokenCacheDir := "./.tokenfiles"
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir, url.QueryEscape("calendar-go.json"))
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile := tokenCacheFile()
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		fmt.Println(err)
	}
	return config.Client(ctx, tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

func get_Events() *calendar.Events {
	ctx := context.Background()
	b, err := ioutil.ReadFile("./config/client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		fmt.Println(err)
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)

	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve calendar Client: %v", err)
	}
	t := time.Now()
	events, err := srv.Events.List("atcoder.jp_gqd1dqpjbld3mhfm4q07e4rops@group.calendar.google.com").ShowDeleted(false).
		SingleEvents(true).TimeMin(t.Format(time.RFC3339)).TimeMax(t.AddDate(0, 0, 1).Add(time.Duration(-t.Hour()) * time.Hour).Format(time.RFC3339)).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events. %v", err)
	}

	return events
}

func post_Slack(text string) {
	json_text := "{" + "text" + ":" + text + "}"

	req, err := http.NewRequest(
		"POST",
		web_hook,
		bytes.NewBuffer([]byte(json_text)),
	)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Context-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
}

func main() {
	events := get_Events()

	if len(events.Items) > 0 {
		text := "今日のコンテスト情報\n"
		for _, i := range events.Items {
			text += fmt.Sprintln(i)
		}
		post_Slack(text + atcoder_url)
	}
}
