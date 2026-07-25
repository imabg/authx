package db

import (
	"context"

	"github.com/imabg/authx/pkg/config"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type DB struct {
	conn *pgx.Conn
}

func Setup(ctx context.Context, config config.ApplicationConfig) (*DB, error) {
	conn, err := pgx.Connect(ctx, config.Database.URI)
	if err != nil {
		zap.L().Error("Error while connecting to postgres", zap.Error(err))
		return nil, err
	}
	defer conn.Close(ctx)
	if err := conn.Ping(ctx); err != nil {
		zap.L().Error("Postgres connection not successful", zap.Error(err))
		return nil, err
	}
	zap.L().Info("postgres server is connected")
	return &DB{
		conn: conn,
	}, nil
}

func (conn *DB) Connection() *pgx.Conn {
	return conn.conn
}
