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
	db      *pgxpool.Pool
	queries *query.Queries
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db:      db,
		queries: query.New(db),
	}
}

func (r *Repo) Get(ctx context.Context, shortURL repository.ShortURL) (repository.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
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

func (r *Repo) Search(ctx context.Context, url repository.URL) (repository.ShortURL, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
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

func (r *Repo) Add(ctx context.Context, shortURL repository.ShortURL, url repository.URL) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
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

func (r *Repo) GetByURLs(ctx context.Context, urls []string) ([]repository.Record, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if len(urls) == 0 {
		return []repository.Record{}, nil
	} else {
		rows, err := r.queries.GetByURLs(ctx, urls)
		if err != nil {
			return nil, fmt.Errorf("psql error GetByURLs: %w", err)
		}
		records := make([]repository.Record, 0, len(rows))
		for _, row := range rows {
			records = append(records, repository.Record{
				ID:       int(row.ID),
				URL:      row.URL,
				ShortURL: row.ShortURL,
			})
		}
		return records, nil
	}
}

func (r *Repo) AddMany(ctx context.Context, records []repository.ArgAddMany) ([]repository.Record, error) {
	if len(records) == 0 {
		return []repository.Record{}, nil
	} else {
		shortURLs := make([]string, 0, len(records))
		urls := make([]string, 0, len(records))
		for _, rec := range records {
			shortURLs = append(shortURLs, string(rec.ShortURL))
			urls = append(urls, string(rec.URL))
		}
		paramsAddMany := query.AddManyParams{
			ShortUrls: shortURLs,
			Urls:      urls,
		}
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		inserts, err := r.queries.AddMany(ctx, paramsAddMany)
		if err != nil {
			return nil, fmt.Errorf("psql error AddMany: %w", err)
		}
		if len(inserts) == 0 {
			return []repository.Record{}, nil
		}
		res := make([]repository.Record, 0, len(inserts))
		for _, insert := range inserts {
			res = append(res, repository.Record{
				ID:       int(insert.ID),
				URL:      repository.URL(insert.URL),
				ShortURL: repository.ShortURL(insert.ShortURL),
			})
		}
		return res, nil
	}
}

// InTx(ctx context.Context, fn func(r Repository) error) error
func (r *Repo) InTx(ctx context.Context, fn func(r repository.Repository) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	txRepo := &Repo{db: r.db, queries: r.queries.WithTx(tx)}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
