package token

import (
	"crypto/rand"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	tokenStr string
	expAt    time.Time
}

func (t *JWT) String() string {
	return t.tokenStr
}

func (t *JWT) ExpAt() time.Time {
	return t.expAt
}

type Opaque struct {
	tokenStr string
	expAt    time.Time
}

func (o *Opaque) String() string {
	return o.tokenStr
}

func (o *Opaque) ExpAt() time.Time {
	return o.expAt
}

func (o *Opaque) Hash() ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(o.tokenStr), 12)
}

type Maker struct {
	SecretKey              string
	Issuer                 string
	AccessTokenExpiration  time.Duration // optional, default is 1 hour
	RefreshTokenExpiration time.Duration // optional, default is 7 days

	jwtv *jwt.Validator
}

func NewMaker(secretKey, issuer string) *Maker {
	return &Maker{
		SecretKey:              secretKey,
		Issuer:                 issuer,
		AccessTokenExpiration:  1 * time.Hour,
		RefreshTokenExpiration: 7 * 24 * time.Hour,

		jwtv: jwt.NewValidator(jwt.WithIssuedAt(), jwt.WithExpirationRequired(), jwt.WithIssuer(issuer)),
	}
}

// SetAccessTokenExpiration sets the expiration time for the tokenStr in hourly format.
// 1 = 1 hour, 12 = 12 hours, etc.
func (m *Maker) SetAccessTokenExpiration(expiration int) {
	m.AccessTokenExpiration = time.Duration(expiration) * time.Hour
}

// SetRefreshTokenExpiration sets the expiration time for the tokenStr in daily format.
// 1 = 1 day, 12 = 12 days, etc.
func (m *Maker) SetRefreshTokenExpiration(expiration int) {
	m.AccessTokenExpiration = time.Duration(expiration) * time.Hour * 24
}

func (m *Maker) NewJWT(userId uint64) (*JWT, error) {
	now := time.Now()
	expAt := now.Add(m.AccessTokenExpiration)

	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expAt),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    m.Issuer,
		Subject:   fmt.Sprintf("%d", userId),
	}

	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(m.SecretKey))
	if err != nil {
		return nil, err
	}

	return &JWT{
		tokenStr: tokenStr,
		expAt:    expAt,
	}, nil
}

// ValidateJWT checks if the tokenStr is valid and returns the user ID.
// It returns an error if the tokenStr is invalid or expired.
func (m *Maker) ValidateJWT(tokenStr string) (uint64, error) {
	token, err := m.validateJWTStr(tokenStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse tokenStr: %w", err)
	}

	claims, err := m.validateJWTClaims(token)
	if err != nil {
		return 0, fmt.Errorf("failed to validate tokenStr claims: %w", err)
	}

	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %w", err)
	}

	return userID, nil
}

// ValidateJWTStr validates the token string and returns the parsed JWT.
// It returns an error if the token string cannot be parsed using the secret key.
func (m *Maker) validateJWTStr(tokenStr string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.SecretKey), nil
	})
	if err != nil {
		return token, err
	}

	return token, nil
}

// ValidateJWTClaims validates the claims of the token.
// it checks for expiration, issued at, and issuer
func (m *Maker) validateJWTClaims(token *jwt.Token) (*jwt.RegisteredClaims, error) {
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, fmt.Errorf("invalid tokenStr claims")
	}

	if err := m.jwtv.Validate(claims); err != nil {
		return nil, fmt.Errorf("tokenStr validation failed: %w", err)
	}

	return claims, nil
}

func (m *Maker) NewOpaque() *Opaque {
	// Generate a random 32-byte string
	return &Opaque{
		tokenStr: rand.Text(),
		expAt:    time.Now().Add(m.RefreshTokenExpiration),
	}
}

func (m *Maker) ValidateOpaque(plain string, hash []byte) (bool, error) {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(plain)); err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func (m *Maker) Tokens(userID uint64) (*JWT, *Opaque, error) {
	accessToken, err := m.NewJWT(userID)
	if err != nil {
		return nil, nil, err
	}

	refreshToken := m.NewOpaque()
	return accessToken, refreshToken, nil
}
