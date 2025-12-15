package main

import (
	"log"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/config"
	handlers "github.com/IvanOplesnin/url-shortener/internal/handler"
	"github.com/IvanOplesnin/url-shortener/internal/logger"
	inmemory "github.com/IvanOplesnin/url-shortener/internal/repository/in_memory"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server run error:%v", err)
	}
}

func run() error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	err = logger.SetupLogger(cfg.Logger.Level, cfg.Logger.Format)
	if err != nil {
		log.Fatal(err)
	}

	baseURL := cfg.BaseURL
	storage := inmemory.NewStorage(cfg.FilePath)
	mux := handlers.InitHandlers(storage, baseURL)
	return http.ListenAndServe(cfg.Server.String(), mux)
}
