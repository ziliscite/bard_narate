package service

import (
	"context"
	"fmt"
	"github.com/ziliscite/bard_narate/auth/internal/domain"
	"github.com/ziliscite/bard_narate/auth/internal/repository"
	"github.com/ziliscite/bard_narate/auth/pkg/token"
	"golang.org/x/oauth2"
	"strings"
)

type OAuthProvider interface {
	// AuthenticationURL returns the authentication URL for the provider.
	AuthenticationURL(state string) string
	// Exchange exchanges the authorization code for an access token.
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	// User retrieves the user information from the provider using the access token.
	User(ctx context.Context, token *oauth2.Token) (*domain.User, error)
}

type OAuthAuthenticator interface {
	AuthenticationURL(provider, state string) (string, error)
	// AuthenticationCallback handles the OAuth callback and returns a token.
	AuthenticationCallback(ctx context.Context, provider, code string) (*domain.Token, error)
}

type OAuthOrchestrator struct {
	github OAuthProvider
	google OAuthProvider

	tr repository.Token
	ur repository.User
	tm *token.Maker
}

func NewOAuthAuthenticator(github, google OAuthProvider, tr repository.Token, ur repository.User, tm *token.Maker) OAuthAuthenticator {
	return &OAuthOrchestrator{
		github: github,
		google: google,
		tr:     tr,
		ur:     ur,
		tm:     tm,
	}
}

func (o *OAuthOrchestrator) AuthenticationURL(provider, state string) (string, error) {
	providerInstance, err := o.getProvider(provider)
	if err != nil {
		return "", err
	}

	return providerInstance.AuthenticationURL(state), nil
}

func (o *OAuthOrchestrator) AuthenticationCallback(ctx context.Context, provider, code string) (*domain.Token, error) {
	providerInstance, err := o.getProvider(provider)
	if err != nil {
		return nil, err
	}

	oauthToken, err := providerInstance.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange OAuth code: %w", err)
	}

	userInfo, err := providerInstance.User(ctx, oauthToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if err = o.getOrCreateUser(ctx, userInfo); err != nil {
		return nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	tkn, err := o.createUserToken(ctx, userInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user token: %w", err)
	}

	return tkn, nil
}

func (o *OAuthOrchestrator) getProvider(provider string) (OAuthProvider, error) {
	switch strings.ToUpper(provider) {
	case "GITHUB":
		return o.github, nil
	case "GOOGLE":
		return o.google, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (o *OAuthOrchestrator) getOrCreateUser(ctx context.Context, userInfo *domain.User) error {
	user, err := o.ur.FindByProviderUserID(ctx, userInfo.Provider, userInfo.ProviderUserID)
	if err != nil {
		return err
	}

	if user != nil {
		userInfo.ID = user.ID
		return nil
	}

	if err = o.ur.Save(ctx, userInfo); err != nil {
		return err
	}

	return nil
}

func (o *OAuthOrchestrator) createUserToken(ctx context.Context, userID uint64) (*domain.Token, error) {
	accessToken, refreshToken, err := o.tm.Tokens(userID)
	if err != nil {
		return nil, err
	}

	refreshTokenHash, err := refreshToken.Hash()
	if err != nil {
		return nil, err
	}

	tkn := domain.NewToken(userID, accessToken.String(), accessToken.ExpAt())
	tkn.SetRefreshToken(refreshToken.String(), refreshTokenHash, refreshToken.ExpAt())

	if err = o.tr.Save(ctx, tkn); err != nil {
		return nil, err
	}

	return tkn, nil
}
