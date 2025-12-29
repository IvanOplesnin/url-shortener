package main

import (
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
	"github.com/jackc/pgx/v5/pgxpool"
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
	persistedRepo, db, err := createRepo(cfg)

	if err != nil {
		logger.Log.Fatalf("Can`t create repository %s", err)
	}
	svc := shortener.New(persistedRepo, baseURL)
	mux := handlers.InitHandlers(svc, baseURL, db)
	return http.ListenAndServe(cfg.Server.String(), mux)
}

func createRepo(cfg *config.Config) (*persisted.Repo, *pgxpool.Pool, error) {
	fileStorage := filestorage.NewJSONStore(cfg.FilePath)
	if cfg.DBDSN != "" {
		db, err := psql.Connect(cfg.DBDSN)
		if err != nil {
			return nil, nil, err
		}
		repo := psql.NewRepo(db)
		persisterdRepo, err := persisted.New(repo, nil, nil, fileStorage, nil)
		if err != nil {
			return nil, nil, err
		}
		return persisterdRepo, db, nil
	}
	repo := inmemory.NewRepo()
	persisterdRepo, err := persisted.New(repo, repo, repo, fileStorage, repo)
	if err != nil {
		return nil, nil, err
	}
	return persisterdRepo, nil, nil
}
