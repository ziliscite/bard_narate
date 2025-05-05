package repository

import (
	"context"
	"github.com/ziliscite/bard_narate/auth/internal/domain"
	"github.com/ziliscite/bard_narate/auth/pkg/github"
	"golang.org/x/oauth2"
	"strconv"
)

type GitHub struct {
	g *github.OAuth
}

func NewGitHub(g *github.OAuth) *GitHub {
	return &GitHub{
		g: g,
	}
}

func (g *GitHub) AuthenticationURL(state string) string {
	return g.g.AuthURL(state)
}

func (g *GitHub) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.g.Exchange(ctx, code)
}

// User retrieves the user information from GitHub using the opaque access token.
func (g *GitHub) User(ctx context.Context, token *oauth2.Token) (*domain.User, error) {
	userInfo, err := g.g.User(ctx, token)
	if err != nil {
		return nil, err
	}

	user := domain.NewUser("GitHub", strconv.Itoa(userInfo.ID), *userInfo.Email, userInfo.Name)
	if userInfo.Login != "" {
		user.SetUsername(userInfo.Login)
	}

	if userInfo.AvatarURL != "" {
		user.SetPicture(userInfo.AvatarURL)
	}

	if userInfo.Login != "" {
		user.SetUsername(userInfo.Login)
	}

	return user, nil
}
