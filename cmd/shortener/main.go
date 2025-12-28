package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/config"
	"github.com/IvanOplesnin/url-shortener/internal/filestorage"
	handlers "github.com/IvanOplesnin/url-shortener/internal/handler"
	"github.com/IvanOplesnin/url-shortener/internal/logger"
	inmemory "github.com/IvanOplesnin/url-shortener/internal/repository/in_memory"
	"github.com/IvanOplesnin/url-shortener/internal/repository/persisted"
	"github.com/IvanOplesnin/url-shortener/internal/repository/psql"
	"github.com/IvanOplesnin/url-shortener/internal/service/shortener"
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
	repo := inmemory.NewRepo()

	var db *sql.DB

	if cfg.DbDSN != "" {
		db, err = psql.Connect(cfg.DbDSN)
		if err != nil {
			logger.Log.Fatalf("Can`t connect to database: %s", err)
		}
		defer db.Close()
	}

	fileStorage := filestorage.NewJSONStore(cfg.FilePath)
	persistedRepo, err := persisted.New(repo, repo, repo, fileStorage, repo)

	if err != nil {
		logger.Log.Fatalf("Can`t create repository %s", err)
	}
	svc := shortener.New(persistedRepo, baseURL)
	mux := handlers.InitHandlers(svc, baseURL, db)
	return http.ListenAndServe(cfg.Server.String(), mux)
}
