package google

import (
	"context"
	"encoding/json"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"net/http"
)

type UserInfo struct {
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Iss           string `json:"iss"`
	Aud           string `json:"aud"`
	AuthTime      int    `json:"auth_time"`
	UserID        string `json:"user_id"`
	Sub           string `json:"sub"`
	Iat           int    `json:"iat"`
	Exp           int    `json:"exp"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Firebase      struct {
		Identities struct {
			GoogleCom []string `json:"google.com"`
			Email     []string `json:"email"`
		} `json:"identities"`
		SignInProvider string `json:"sign_in_provider"`
	} `json:"firebase"`
}

type OAuth struct {
	cfg *oauth2.Config
}

func NewOAuthClient(clientId, clientSecret, redirectUrl string, scopes ...string) *OAuth {
	return &OAuth{
		cfg: &oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			Scopes:       scopes,
			Endpoint:     endpoints.Google,
			RedirectURL:  redirectUrl,
		},
	}
}

// AuthURL generates the URL for the Google OAuth2 authorization request.
func (o *OAuth) AuthURL(state string) string {
	//o.cfg.RedirectURL = "http://localhost:3002/oauth2/google/callback"
	return o.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange exchanges the authorization code for an access token. Is a callback function that is supposed to be called by Google.
func (o *OAuth) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return o.cfg.Exchange(ctx, code)
}

// Client returns an HTTP client that uses the provided token for authentication.
func (o *OAuth) Client(ctx context.Context, token *oauth2.Token) *http.Client {
	return o.cfg.Client(ctx, token)
}

func (o *OAuth) Config() *oauth2.Config {
	return o.cfg
}

// User retrieves the user information from Google using the JWT access token.
func (o *OAuth) User(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := o.cfg.Client(ctx, token)

	userInfoResp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}

	var user UserInfo
	if err = json.NewDecoder(userInfoResp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}
