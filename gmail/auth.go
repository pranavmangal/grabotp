package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pranavmangal/grabotp/config"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	keyringService = "grabotp"
	redirectURL    = "http://localhost:8080"
)

func getOauthConfig(clientId string) oauth2.Config {
	return oauth2.Config{
		ClientID:    clientId,
		Scopes:      []string{gmail.GmailReadonlyScope},
		Endpoint:    google.Endpoint,
		RedirectURL: redirectURL,
	}
}

// GetGmailService retrieves a service to use the Gmail API
func GetGmailService(user string) *gmail.Service {
	clientId, err := config.ReadClientId()
	if err != nil {
		log.Fatalf("Unable to read client ID: %v", err)
	}

	conf := getOauthConfig(clientId)
	client := getClient(&conf, user)
	srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	return srv
}

// Retrieves or fetches a token and returns the client.
func getClient(config *oauth2.Config, user string) *http.Client {
	tok, err := tokenFromKeyring(user)
	if err != nil {
		fmt.Printf("Could not find token for %s, re-authenticating...\n", user)
		tok = getTokenFromWeb(config)
		saveTokenToKeyring(user, tok)
	}

	return config.Client(context.Background(), tok)
}

// Requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	verifier := oauth2.GenerateVerifier()
	authURL := config.AuthCodeURL(
		"state-token",
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)

	authCodeChan := make(chan string)
	errChan := make(chan error)

	mux := http.NewServeMux()
	server := &http.Server{Addr: ":8080", Handler: mux}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "state-token" {
			http.Error(w, "Invalid state token", http.StatusBadRequest)
			errChan <- fmt.Errorf("invalid state token")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing auth code", http.StatusBadRequest)
			errChan <- fmt.Errorf("missing auth code")
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Success! You can close this window."))
		authCodeChan <- code

		go server.Shutdown(context.Background())
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	fmt.Printf("Go to the following link in your browser to authorize the application: \n\n%v\n\n", authURL)

	var authCode string
	select {
	case code := <-authCodeChan:
		authCode = code
	case err := <-errChan:
		log.Fatalf("Unable to read authorization code: %v", err)
	case <-time.After(5 * time.Minute):
		log.Fatalf("timed out waiting for authorization code")
	}

	tok, err := config.Exchange(context.TODO(), authCode, oauth2.VerifierOption(verifier))
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	return tok
}

// Retrieves a token from the system keychain for a specific user.
func tokenFromKeyring(user string) (*oauth2.Token, error) {
	secret, err := keyring.Get(keyringService, user)
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

// Saves a token to the system keychain for a specific user.
func saveTokenToKeyring(user string, token *oauth2.Token) {
	fmt.Println("Saving credential to system keychain...")
	b, err := json.Marshal(token)
	if err != nil {
		log.Fatalf("Unable to marshal token to JSON: %v", err)
	}

	err = keyring.Set(keyringService, user, string(b))
	if err != nil {
		log.Fatalf("Unable to save token to keychain: %v", err)
	}

	fmt.Println("Successfully saved credential!")
}

func AddAccount() {
	clientId, err := config.ReadClientId()
	if err != nil {
		log.Fatalf("Unable to read client ID: %v", err)
	}

	if clientId == "" {
		fmt.Printf("Please enter your client ID: \n")
		if _, err := fmt.Scan(&clientId); err != nil {
			log.Fatalf("Unable to read client ID: %v", err)
		}

		err = config.WriteClientId(clientId)
		if err != nil {
			log.Fatalf("Unable to write client ID to config: %v", err)
		}
	}

	conf := getOauthConfig(clientId)
	tok := getTokenFromWeb(&conf)
	client := conf.Client(context.Background(), tok)
	srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	profile, err := srv.Users.GetProfile("me").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve user profile: %v", err)
	}

	user := profile.EmailAddress
	fmt.Printf("Successfully authenticated as %s!\n", user)

	err = config.AddAccount(user)
	if err != nil {
		log.Fatalf("Unable to add account to config: %v", err)
	}

	saveTokenToKeyring(user, tok)
}

func DeleteToken(user string) error {
	return keyring.Delete(keyringService, user)
}
