package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func UploadDBBackupToDrive(_ []byte, dbPath string) (string, error) {
	ctx := context.Background()

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("CLIENT_ID and CLIENT_SECRET are required for OAuth2")
	}

	tokenFile := os.Getenv("OAUTH_TOKEN_FILE")
	if tokenFile == "" {
		tokenFile = "./drive_token.json"
	}

	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{drive.DriveFileScope},
		RedirectURL:  "http://localhost:8080/oauth2callback",
	}

	var token *oauth2.Token
	if _, err := os.Stat(tokenFile); err == nil {
		f, err := os.Open(tokenFile)
		if err == nil {
			json.NewDecoder(f).Decode(&token)
			f.Close()
		}
	}

	if token == nil {
		state := "state-token"
		codeCh := make(chan string)
		registerOAuthCallback(state, codeCh)

		authURL := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
		fmt.Printf("Visit the URL for the auth dialog: %s\n", authURL)
		code := <-codeCh

		tok, err := conf.Exchange(ctx, code)
		if err != nil {
			return "", err
		}
		token = tok
		f, err := os.Create(tokenFile)
		if err == nil {
			json.NewEncoder(f).Encode(token)
			f.Close()
		}
	}

	client := conf.Client(ctx, token)
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("failed to create drive service: %v", err)
	}

	tmp := filepath.Join(os.TempDir(), filepath.Base(dbPath)+"."+time.Now().Format("20060102-150405")+".bak")
	in, err := os.Open(dbPath)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	out.Close()

	f, err := os.Open(tmp)
	if err != nil {
		return "", err
	}
	defer f.Close()

	folder := os.Getenv("FOLDER_ID")
	if folder == "" {
		return "", fmt.Errorf("FOLDER_ID not set")
	}
	file := &drive.File{Name: filepath.Base(tmp), Parents: []string{folder}}
	uploaded, err := srv.Files.Create(file).Media(f).Do()
	if err != nil {
		return "", err
	}
	return uploaded.Id, nil
}

var (
	oauthCallbackChans = make(map[string]chan string)
)

func registerOAuthCallback(state string, ch chan string) {
	oauthCallbackChans[state] = ch
}

func HandleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	ch, ok := oauthCallbackChans[state]
	if !ok {
		http.Error(w, "no pending oauth request", http.StatusBadRequest)
		return
	}
	if r.FormValue("state") != state {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	code := r.FormValue("code")
	fmt.Fprint(w, "ok")
	ch <- code
	delete(oauthCallbackChans, state)
}
