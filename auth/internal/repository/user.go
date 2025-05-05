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

type User interface {
	// Save saves a new user to the database.
	Save(ctx context.Context, user *domain.User) error
	// Find retrieves a user by ID. Used for user lookup post-authentication (through JWT).
	Find(ctx context.Context, id uint64) (*domain.User, error)
	// FindByProviderUserID check if a user exists in the database.
	FindByProviderUserID(ctx context.Context, provider, providerUserID string) (*domain.User, error)
	// Delete removes a user from the database.
	Delete(ctx context.Context, id uint64) error
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) (User, error) {
	if db == nil {
		return nil, ErrNilPool
	}

	return &userRepository{db: db}, nil
}

func (r *userRepository) Save(ctx context.Context, user *domain.User) error {
	args := []interface{}{
		user.Provider, user.ProviderUserID, user.Picture, user.Email, user.Name, user.Username,
	}

	if err := r.db.QueryRow(ctx, `
		INSERT INTO users (provider, provider_user_id, picture, email, name, username)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, args).Scan(&user.ID); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("user already exists: %w", ErrDuplicate)
		case errors.As(err, &pgErr) && pgErr.Code == "23503":
			return fmt.Errorf("%w provider: %w", ErrInvalid, err)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}

func (r *userRepository) Find(ctx context.Context, id uint64) (*domain.User, error) {
	var user domain.User

	if err := r.db.QueryRow(ctx, `
		SELECT 
			id, provider, provider_user_id, picture, email, 
			name, username, created_at, updated_at, version
		FROM tokens
		WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Provider, &user.ProviderUserID, &user.Picture, &user.Email,
		&user.Name, &user.Username, &user.CreatedAt, &user.UpdatedAt, &user.Version,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, fmt.Errorf("user %w: %w", ErrNotFound, err)
		default:
			return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return &user, nil
}

func (r *userRepository) FindByProviderUserID(ctx context.Context, provider, providerUserID string) (*domain.User, error) {
	var user domain.User

	if err := r.db.QueryRow(ctx, `
		SELECT 
			id, provider, provider_user_id, picture, email, 
			name, username, created_at, updated_at, version
		FROM tokens
		WHERE provider = $1 AND provider_user_id = $2
	`, provider, providerUserID).Scan(
		&user.ID, &user.Provider, &user.ProviderUserID, &user.Picture, &user.Email,
		&user.Name, &user.Username, &user.CreatedAt, &user.UpdatedAt, &user.Version,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, nil
		default:
			return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return &user, nil
}

func (r *userRepository) Delete(ctx context.Context, id uint64) error {
	if _, err := r.db.Exec(ctx, `
		DELETE FROM users
		WHERE id = $1
	`, id); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("user %w: %w", ErrNotFound, err)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}
