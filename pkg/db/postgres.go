package db

import (
	"context"

	"github.com/imabg/authx/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DB struct {
	pool *pgxpool.Pool
}

func Setup(ctx context.Context, config config.ApplicationConfig) (*DB, error) {
	pool, err := pgxpool.New(ctx, config.Database.URI)
	if err != nil {
		zap.L().Error("Error while connecting to postgres", zap.Error(err))
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		zap.L().Error("Postgres connection not successful", zap.Error(err))
		return nil, err
	}
	zap.L().Info("postgres server is connected")
	return &DB{
		pool: pool,
	}, nil
}

func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

func (d *DB) Close() {
	if d != nil && d.pool != nil {
		d.pool.Close()
	}
}
