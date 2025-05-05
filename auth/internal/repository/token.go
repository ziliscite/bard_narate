package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/ziliscite/bard_narate/auth/internal/domain"
)

//CREATE TABLE IF NOT EXISTS tokens (
//	id SERIAL PRIMARY KEY,
//	user_id INTEGER NOT NULL,
//	access_token VARCHAR(255) NOT NULL,
//	refresh_token VARCHAR(255) DEFAULT NULL,
//	scopes VARCHAR(1024) NOT NULL,
//	access_token_expires_at TIMESTAMP NOT NULL,
//	refresh_token_expires_at TIMESTAMP DEFAULT NULL,
//	created_at TIMESTAMP DEFAULT NOW(),
//	updated_at TIMESTAMP DEFAULT NOW(),
//	version INTEGER NOT NULL DEFAULT 1,
//	FOREIGN KEY (user_id) REFERENCES users(id),
//	UNIQUE (user_id, access_token)
//);

type Token interface {
	Find(ctx context.Context, accessToken string) (*domain.Token, error)
	// Save saves a new token to the database.
	Save(ctx context.Context, token *domain.Token) error
	Update(ctx context.Context, tokenID uint64, token *domain.Token) error
	Delete(ctx context.Context, tokenID uint64) error
}

type tokenRepository struct {
	db *pgxpool.Pool
}

func NewTokenRepository(db *pgxpool.Pool) (Token, error) {
	if db == nil {
		return nil, ErrNilPool
	}

	return &tokenRepository{db: db}, nil
}

func (r *tokenRepository) Find(ctx context.Context, accessToken string) (*domain.Token, error) {
	var token domain.Token

	var hash []byte
	if err := r.db.QueryRow(ctx, `
		SELECT 
			id, user_id, access_token, refresh_token, scopes, access_token_expires_at,
			refresh_token_expires_at, revoked, created_at, updated_at, revoked_at, version
		FROM tokens
		WHERE access_token = $1
	`, accessToken).Scan(
		&token.ID, &token.UserID, &token.AccessToken,
		&hash, &token.AccessTokenExpiresAt,
		&token.RefreshTokenExpiresAt, &token.Revoked,
		&token.CreatedAt, &token.UpdatedAt, &token.RevokedAt, &token.Version,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, fmt.Errorf("token %w: %w", ErrNotFound, err)
		default:
			return nil, fmt.Errorf("%w while updating token: %w", ErrUnknown, err)
		}
	}

	token.SetRefreshTokenHash(hash)
	return &token, nil
}

func (r *tokenRepository) Save(ctx context.Context, token *domain.Token) error {
	args := []interface{}{
		token.UserID, token.AccessToken, token.RefreshTokenHash(),
		token.AccessTokenExpiresAt, token.RefreshTokenExpiresAt,
	}

	if _, err := r.db.Exec(ctx, `
		INSERT INTO tokens (
			user_id, access_token, refresh_token,
			access_token_expires_at, refresh_token_expires_at
		) VALUES ($1, $2, $3, $4, $5)
	`, args); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("token already exists: %w", ErrDuplicate)
		default:
			return fmt.Errorf("%w while updating token: %w", ErrUnknown, err)
		}
	}

	return nil
}

func (r *tokenRepository) Update(ctx context.Context, tokenID uint64, token *domain.Token) error {
	args := []interface{}{
		token.AccessToken, token.RefreshTokenHash(),
		token.AccessTokenExpiresAt, token.RefreshTokenExpiresAt,
		token.Revoked, token.RevokedAt, tokenID, token.Version,
	}

	if _, err := r.db.Exec(ctx, `
		UPDATE tokens
		SET access_token = $1, refresh_token = $2,
			access_token_expires_at = $3, refresh_token_expires_at = $4, 
			revoked = $5, updated_at = NOW(), revoked_at = $6, version = version + 1
		WHERE id = $7 AND version = $8
	`, args); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("token %w: %w", ErrNotFound, err)
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("token already exists: %w", ErrDuplicate)
		default:
			return fmt.Errorf("%w while updating token: %w", ErrUnknown, err)
		}
	}

	return nil
}

func (r *tokenRepository) Delete(ctx context.Context, tokenID uint64) error {
	if _, err := r.db.Exec(ctx, `
		DELETE FROM tokens
		WHERE id = $1
	`, tokenID); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("token %w: %w", ErrNotFound, err)
		default:
			return fmt.Errorf("%w while updating token: %w", ErrUnknown, err)
		}
	}

	return nil
}
