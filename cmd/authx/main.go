package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"time"

	"github.com/imabg/authx/pkg/config"
	"github.com/imabg/authx/pkg/db"
	"github.com/imabg/authx/pkg/logger"
	"github.com/imabg/authx/server"
	"go.uber.org/zap"
)

func main() {
	env := config.NewConfig()
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "")
	flag.Parse()

	logger.Setup(env.IsDevelopment())
	zap.L().Info("logger is setup", zap.String("env", env.App.ENV), zap.Bool("dev", env.IsDevelopment()))

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()
	database, err := db.Setup(dbCtx, env)
	if err != nil {
		os.Exit(1)
	}
	defer database.Close()

	if err := db.RunMigrations(env); err != nil {
		os.Exit(1)
	}

	srv := server.Setup(env, database)
	srv.Run()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	srv.Close(ctx)
	os.Exit(0)
}
