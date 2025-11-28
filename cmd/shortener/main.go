package main

import (
	"net/http"

	handlers "github.com/IvanOplesnin/url-shortener/internal/handler"
	inmemory "github.com/IvanOplesnin/url-shortener/internal/repository/in_memory"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	baseURL := `http://localhost:8080/`
	storage := inmemory.NewStorage()
	mux := handlers.InitHandlers(storage, baseURL)
	return http.ListenAndServe(`:8080`, mux)
}
