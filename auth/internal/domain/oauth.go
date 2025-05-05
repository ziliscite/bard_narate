package domain

import (
	"strings"
	"time"
)

type User struct {
	ID             uint64 `json:"id"`
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`

	Picture  *string `json:"picture"`
	Email    string  `json:"email"`
	Name     string  `json:"name"`
	Username *string `json:"username,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
}

func NewUser(provider, providerUserID, email, name string) *User {
	return &User{
		Provider:       strings.ToUpper(provider),
		ProviderUserID: providerUserID,
		Email:          email,
		Name:           name,
	}
}

func (u *User) SetPicture(picture string) {
	u.Picture = &picture
}

func (u *User) SetUsername(username string) {
	u.Username = &username
}

type Token struct {
	ID     uint64 `json:"id"`
	UserID uint64 `json:"user_id"`

	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	refreshTokenHash      []byte
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	Revoked               bool      `json:"revoked"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	RevokedAt time.Time `json:"revoked_at"`
	Version   int       `json:"version"`
}

func NewToken(userID uint64, accessToken string, accessTokenExpiresAt time.Time) *Token {
	return &Token{
		UserID:               userID,
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
	}
}

// SetRefreshToken set the optional refresh token and its expiration
func (t *Token) SetRefreshToken(refreshToken string, refreshTokenHash []byte, refreshTokenExpiresAt time.Time) {
	t.RefreshToken = refreshToken
	t.refreshTokenHash = refreshTokenHash
	t.RefreshTokenExpiresAt = refreshTokenExpiresAt
}

func (t *Token) RefreshTokenHash() []byte {
	return t.refreshTokenHash
}

func (t *Token) SetRefreshTokenHash(hash []byte) {
	t.refreshTokenHash = hash
}

func (t *Token) Revoke() {
	t.Revoked = true
	t.RevokedAt = time.Now()
}
