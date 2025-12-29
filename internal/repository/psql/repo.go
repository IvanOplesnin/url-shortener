package psql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IvanOplesnin/url-shortener/internal/logger"
	"github.com/IvanOplesnin/url-shortener/internal/repository"
	"github.com/IvanOplesnin/url-shortener/internal/repository/psql/query"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	queries *query.Queries
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		queries: query.New(db),
	}
}

func (r *Repo) Get(shortURL repository.ShortURL) (repository.URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	url, err := r.queries.Get(ctx, shortURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return repository.URL(""), repository.ErrNotFoundShortURL
	}
	if err != nil {
		return repository.URL(""), fmt.Errorf("psql error Get: %w", err)
	}
	return repository.URL(url), nil
}

func (r *Repo) Search(url repository.URL) (repository.ShortURL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	shortURL, err := r.queries.Search(ctx, url)
	if errors.Is(err, pgx.ErrNoRows) {
		return repository.ShortURL(shortURL), repository.ErrNotFoundURL
	}
	if err != nil {
		return repository.ShortURL(""), fmt.Errorf("psql error Search: %w", err)
	}
	return repository.ShortURL(shortURL), nil
}

func (r *Repo) Add(shortURL repository.ShortURL, url repository.URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	params := query.AddParams{ShortURL: shortURL, URL: url}
	if err := r.queries.Add(ctx, params); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "alias_url_short_url_uk":
				return fmt.Errorf("%w: %v", repository.ErrShortURLAlreadyExists, shortURL)
			case "alias_url_url_uk":
				return fmt.Errorf("%w: %v", repository.ErrAlreadyExists, url)
			default:
				return fmt.Errorf("%w", repository.ErrAlreadyExists)
			}
		}

		return fmt.Errorf("psql error Add: %w", err)
	}
	return nil
}

func (r *Repo) Snapshot() []repository.Record {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rows, err := r.queries.GetAllRecords(ctx)
	if err != nil {
		logger.Log.Errorf("error psql GetAllRecords: %s", err)
		return []repository.Record{}
	}
	recs := make([]repository.Record, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, repository.Record{
			ID:       int(r.ID),
			URL:      r.URL,
			ShortURL: r.ShortURL,
		})
	}
	return recs
}
