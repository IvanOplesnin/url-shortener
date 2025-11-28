package main

import (
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/config"
	handlers "github.com/IvanOplesnin/url-shortener/internal/handler"
	inmemory "github.com/IvanOplesnin/url-shortener/internal/repository/in_memory"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg := config.ParseFlags()
	baseURL := cfg.BaseURL
	storage := inmemory.NewStorage()
	mux := handlers.InitHandlers(storage, baseURL)
	return http.ListenAndServe(cfg.String(), mux)
}
