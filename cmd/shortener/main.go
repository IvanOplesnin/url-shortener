package main

import (
	"database/sql"
	"errors"
	"fmt"
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
	migrate "github.com/IvanOplesnin/url-shortener/migrations"
	"github.com/jackc/pgx/v5/pgconn"
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
	if err := runMigrate(cfg); err != nil {
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
		persisterdRepo, err := persisted.New(repo, nil, nil, fileStorage, nil, repo, repo)
		if err != nil {
			return nil, nil, err
		}
		return persisterdRepo, db, nil
	}
	repo := inmemory.NewRepo()
	persisterdRepo, err := persisted.New(repo, repo, repo, fileStorage, repo, nil, repo)
	if err != nil {
		return nil, nil, err
	}
	return persisterdRepo, nil, nil
}

func runMigrate(cfg *config.Config) error {
	if cfg.DBDSN == "" {
		return nil
	}
	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := migrate.Up(db); err != nil {
		if isInsufficientPrivilege(err) {
			logger.Log.Infof("migrations skipped (insufficient privileges): %v", err)
			return nil
		}
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

func isInsufficientPrivilege(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 42501 = insufficient_privilege
		return pgErr.Code == "42501"
	}
	return false
}
