package postgres

import (
	"context"
	"errors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func AutoMigrate(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return err
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func RunInTx(ctx context.Context, db *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	if err = fn(tx); err == nil {
		return tx.Commit(ctx)
	}

	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
		return errors.Join(err, rollbackErr)
	}

	return err
}

func RunInTxReturn[T any](ctx context.Context, db *pgxpool.Pool, fn func(tx pgx.Tx) (*T, error)) (*T, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}

	if res, err := fn(tx); err == nil {
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}

		return res, nil
	}

	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
		return nil, errors.Join(err, rollbackErr)
	}

	return nil, err
}
