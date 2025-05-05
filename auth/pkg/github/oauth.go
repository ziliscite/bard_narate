package github

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"net/http"
	"strings"
	"time"
)

type UserInfo struct {
	Login                   string    `json:"login"`
	ID                      int       `json:"id"`
	NodeID                  string    `json:"node_id"`
	AvatarURL               string    `json:"avatar_url"`
	GravatarID              string    `json:"gravatar_id"`
	URL                     string    `json:"url"`
	HTMLURL                 string    `json:"html_url"`
	FollowersURL            string    `json:"followers_url"`
	FollowingURL            string    `json:"following_url"`
	GistsURL                string    `json:"gists_url"`
	StarredURL              string    `json:"starred_url"`
	SubscriptionsURL        string    `json:"subscriptions_url"`
	OrganizationsURL        string    `json:"organizations_url"`
	ReposURL                string    `json:"repos_url"`
	EventsURL               string    `json:"events_url"`
	ReceivedEventsURL       string    `json:"received_events_url"`
	Type                    string    `json:"type"`
	UserViewType            string    `json:"user_view_type"`
	SiteAdmin               bool      `json:"site_admin"`
	Name                    string    `json:"name"`
	Company                 *string   `json:"company"`
	Blog                    string    `json:"blog"`
	Location                *string   `json:"location"`
	Email                   *string   `json:"email"`
	Hireable                *bool     `json:"hireable"`
	Bio                     string    `json:"bio"`
	TwitterUsername         *string   `json:"twitter_username"`
	NotificationEmail       *string   `json:"notification_email"`
	PublicRepos             int       `json:"public_repos"`
	PublicGists             int       `json:"public_gists"`
	Followers               int       `json:"followers"`
	Following               int       `json:"following"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	PrivateGists            int       `json:"private_gists"`
	TotalPrivateRepos       int       `json:"total_private_repos"`
	OwnedPrivateRepos       int       `json:"owned_private_repos"`
	DiskUsage               int       `json:"disk_usage"`
	Collaborators           int       `json:"collaborators"`
	TwoFactorAuthentication bool      `json:"two_factor_authentication"`
	Plan                    struct {
		Name          string `json:"name"`
		Space         int    `json:"space"`
		Collaborators int    `json:"collaborators"`
		PrivateRepos  int    `json:"private_repos"`
	} `json:"plan"`
}

type EmailEntry struct {
	Email      string  `json:"email"`
	Primary    bool    `json:"primary"`
	Verified   bool    `json:"verified"`
	Visibility *string `json:"visibility"`
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
			Endpoint:     endpoints.GitHub,
			RedirectURL:  redirectUrl,
		},
	}
}

// AuthURL generates the URL for the GitHub OAuth2 authorization request.
func (o *OAuth) AuthURL(state string) string {
	return o.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange exchanges the authorization code for an access token. Is a callback function that is supposed to be called by GitHub.
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

// User retrieves the user information from GitHub using the opaque access token.
func (o *OAuth) User(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := o.cfg.Client(ctx, token)

	userInfoResp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}

	var user UserInfo
	if err = json.NewDecoder(userInfoResp.Body).Decode(&user); err != nil {
		return nil, err
	}

	if user.Email != nil {
		return &user, nil
	}

	scopes := userInfoResp.Header.Get("X-OAuth-Scopes")
	if scopes == "" {
		return nil, fmt.Errorf("no scopes found in response header")
	}

	emailAuth := strings.Contains(scopes, "user") || strings.Contains(scopes, "read:user") || strings.Contains(scopes, "user:email")
	if !emailAuth {
		return nil, fmt.Errorf("email scope not granted")
	}

	emailsResp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return nil, err
	}
	defer emailsResp.Body.Close()

	var emails []EmailEntry
	if err = json.NewDecoder(emailsResp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	for _, email := range emails {
		if !email.Primary {
			continue
		}

		user.Email = &email.Email
		break
	}

	if user.Email == nil {
		return nil, fmt.Errorf("no primary email found")
	}

	return &user, nil
}
