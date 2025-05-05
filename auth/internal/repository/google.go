package repository

import (
	"context"
	"github.com/ziliscite/bard_narate/auth/internal/domain"
	"github.com/ziliscite/bard_narate/auth/pkg/google"
	"golang.org/x/oauth2"
)

type Google struct {
	g *google.OAuth
}

func NewGoogle(g *google.OAuth) *Google {
	return &Google{
		g: g,
	}
}

func (g *Google) AuthenticationURL(state string) string {
	return g.g.AuthURL(state)
}

func (g *Google) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.g.Exchange(ctx, code)
}

// User retrieves the user information from Google using a JWT access token.
func (g *Google) User(ctx context.Context, token *oauth2.Token) (*domain.User, error) {
	userInfo, err := g.g.User(ctx, token)
	if err != nil {
		return nil, err
	}

	user := domain.NewUser("Google", userInfo.Sub, userInfo.Email, userInfo.Name)
	user.SetPicture(userInfo.Picture)

	return user, nil
}
