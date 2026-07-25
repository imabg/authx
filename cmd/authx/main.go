package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"time"

	"github.com/imabg/authx/pkg/config"
	"github.com/imabg/authx/pkg/logger"
	"github.com/imabg/authx/server"
	"go.uber.org/zap"
)


func main() {
	env := config.NewConfig()
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second * 15 , "")
	flag.Parse()
	// setup logger
	logger.Setup()
	zap.L().Info("logger is setup")	

	// setup server and graceful shutdown
	srv := server.Setup(env)
	srv.Run()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<- c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	srv.Close(ctx)
	os.Exit(0)
}
