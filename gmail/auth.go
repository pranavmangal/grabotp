package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "grabotp"
	keyringUser    = "gmail_token"
)

// GetClient retrieves a client to use the Gmail API
func GetClient() *gmail.Service {
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := getClient(config)
	srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	return srv
}

// Retrieves or fetches a token and returns the client.
func getClient(config *oauth2.Config) *http.Client {
	tok, err := tokenFromKeyring()
	if err != nil {
		tok = getTokenFromWeb(config)
		saveTokenToKeyring(tok)
	}

	return config.Client(context.Background(), tok)
}

// Requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	return tok
}

// Retrieves a token from the system keychain.
func tokenFromKeyring() (*oauth2.Token, error) {
	secret, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return nil, err
	}

	tok := &oauth2.Token{}
	err = json.Unmarshal([]byte(secret), tok)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal token from keychain: %w", err)
	}

	return tok, nil
}

// Saves a token to the system keychain.
func saveTokenToKeyring(token *oauth2.Token) {
	fmt.Println("Saving credential to system keychain...")
	b, err := json.Marshal(token)
	if err != nil {
		log.Fatalf("Unable to marshal token to JSON: %v", err)
	}

	err = keyring.Set(keyringService, keyringUser, string(b))
	if err != nil {
		log.Fatalf("Unable to save token to keychain: %v", err)
	}

	fmt.Println("Successfully saved credential.")
}
