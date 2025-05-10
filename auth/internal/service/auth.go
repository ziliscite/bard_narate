package service

import (
	"context"
	"github.com/ziliscite/bard_narate/auth/internal/domain"
	"github.com/ziliscite/bard_narate/auth/internal/repository"
	"github.com/ziliscite/bard_narate/auth/pkg/token"
	"time"
)

type ServerAuthenticator interface {
	// Authenticate authenticates the user using a JWT access token.
	Authenticate(ctx context.Context, accessToken string) (*domain.User, error)
	// Refresh refreshes the access token using past JWT access token and an opaque refresh token.
	Refresh(ctx context.Context, accessToken, refreshToken string) (*domain.Token, error)
	// Revoke revokes the access token and refresh token.
	Revoke(ctx context.Context, token string) error
}

type Authenticator struct {
	tr repository.Token
	ur repository.User
	tm *token.Maker
}

func NewAuthenticator(tr repository.Token, ur repository.User, tm *token.Maker) ServerAuthenticator {
	return &Authenticator{
		tr: tr,
		ur: ur,
		tm: tm,
	}
}

func (a *Authenticator) Authenticate(ctx context.Context, accessToken string) (*domain.User, error) {
	userID, err := a.tm.ValidateJWT(accessToken)
	if err != nil {
		return nil, err
	}

	user, err := a.ur.Find(ctx, userID)
	if err != nil {
		return nil, err
	}

	// check if

	return user, nil
}

func (a *Authenticator) Refresh(ctx context.Context, accessTokenStr, refreshTokenStr string) (*domain.Token, error) {
	tkn, err := a.validateTokens(ctx, accessTokenStr, refreshTokenStr)
	if err != nil {
		return nil, err
	}

	if err = a.issueTokens(tkn); err != nil {
		return nil, err
	}

	// save the updated token to the database
	if err = a.tr.Update(ctx, tkn.ID, tkn); err != nil {
		return nil, err
	}

	return tkn, nil
}

func (a *Authenticator) validateTokens(ctx context.Context, accessTokenStr, refreshTokenStr string) (*domain.Token, error) {
	// get a token instance from db using an access token string
	tkn, err := a.tr.Find(ctx, accessTokenStr)
	if err != nil {
		return nil, err
	}

	// check if the token is revoked
	if tkn.Revoked {
		return nil, ErrTokenRevoked
	}

	// check if the refresh token is expired
	if tkn.RefreshTokenExpiresAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}

	// validate the refresh token string with its hash from the database
	ok, err := a.tm.ValidateOpaque(refreshTokenStr, tkn.RefreshTokenHash())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidToken
	}

	return tkn, nil
}

func (a *Authenticator) issueTokens(tkn *domain.Token) error {
	// issue new token
	accessToken, refreshToken, err := a.tm.Tokens(tkn.UserID)
	if err != nil {
		return err
	}

	refreshTokenHash, err := refreshToken.Hash()
	if err != nil {
		return err
	}

	tkn.AccessToken = accessToken.String()
	tkn.AccessTokenExpiresAt = accessToken.ExpAt()
	tkn.RefreshToken = refreshToken.String()
	tkn.SetRefreshTokenHash(refreshTokenHash)
	// don't update the refresh token expiration time

	return nil
}

func (a *Authenticator) Revoke(ctx context.Context, token string) error {
	// get a token instance from db using an access token string
	tkn, err := a.tr.Find(ctx, token)
	if err != nil {
		return err
	}

	// check if the token is revoked
	if tkn.Revoked {
		return ErrTokenRevoked
	}

	// revoke the token
	tkn.Revoke()
	return a.tr.Update(ctx, tkn.ID, tkn)
}
