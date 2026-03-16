package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/ar4ie13/gophkeeper/internal/logger"
	"github.com/ar4ie13/gophkeeper/internal/server/auth"
	"github.com/ar4ie13/gophkeeper/internal/server/config"
	"github.com/ar4ie13/gophkeeper/internal/server/handlers"
	"github.com/ar4ie13/gophkeeper/internal/server/repository"
	"github.com/ar4ie13/gophkeeper/internal/server/service"
)

// Build information variables set during compilation via ldflags.
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

// main starts run functions which contains all objects initialization
func main() {
	fmt.Println("Build version: " + buildVersion)
	fmt.Println("Build date: " + buildDate)
	fmt.Println("Build commit: " + buildCommit)
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

// Run initializes all application components and starts the server.
func Run() error {
	cfg := config.NewConfig()
	zlog := logger.NewLogger(cfg.LogConf.Level)
	repo, err := repository.NewRepository(context.Background(), cfg.PGConf, zlog.Logger)
	if err != nil {
		log.Fatal(err)
	}
	service := service.NewService(repo, zlog.Logger, cfg.ServiceConf)
	auth := auth.NewAuth(cfg.AuthConf)
	server := handlers.NewHandler(cfg.ServerConf, zlog.Logger, auth, service)
	if err := server.StartServer(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
