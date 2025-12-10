package main

import (
	"log"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/config"
	handlers "github.com/IvanOplesnin/url-shortener/internal/handler"
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
	baseURL := cfg.BaseURL
	storage := inmemory.NewStorage()
	mux := handlers.InitHandlers(storage, baseURL)
	return http.ListenAndServe(cfg.String(), mux)
}
