package shortener

import (
	"context"
	"errors"
	"fmt"

	"github.com/IvanOplesnin/url-shortener/internal/model"
	"github.com/IvanOplesnin/url-shortener/internal/repository"
	usvc "github.com/IvanOplesnin/url-shortener/internal/service/url"
)

type Service struct {
	r       repository.Repository
	baseURL string
}

type Result struct {
	Short  repository.ShortURL
	Link   string
	Exists bool
}

func New(r repository.Repository, baseURL string) *Service {
	return &Service{r: r, baseURL: baseURL}
}

func (s *Service) Shorten(ctx context.Context, u repository.URL) (Result, error) {
	if _, err := usvc.ParseURL(string(u)); err != nil {
		return Result{}, fmt.Errorf("invalid url: %w", err)
	}

	short, err := s.r.Search(ctx, u)
	if err == nil {
		link, err := usvc.CreateURL(s.baseURL, short)
		if err != nil {
			return Result{}, err
		}
		return Result{Short: short, Link: link, Exists: true}, nil
	}

	if !errors.Is(err, repository.ErrNotFoundURL) {
		return Result{}, err
	}

	short, err = usvc.AddRandomString(ctx, s.r, u)
	if err != nil {
		return Result{}, err
	}

	link, err := usvc.CreateURL(s.baseURL, short)

	if err != nil {
		return Result{}, err
	}
	return Result{Short: short, Link: link, Exists: false}, nil
}

func (s *Service) Resolve(ctx context.Context, short repository.ShortURL) (repository.URL, error) {
	return s.r.Get(ctx, short)
}

func (s *Service) AddRandomString(ctx context.Context, u repository.URL) (repository.ShortURL, error) {
	const retry = 6

	for i := 0; i < retry; i++ {
		short := usvc.GenerateShort(6)

		err := s.r.Add(ctx, short, u)
		if err == nil {
			return short, nil
		}

		if errors.Is(err, repository.ErrShortURLAlreadyExists) || errors.Is(err, repository.ErrAlreadyExists) {
			continue
		}

		return "", err
	}

	return "", fmt.Errorf("can't generate unique short url after %d retries", retry)
}

func (s *Service) Batch(ctx context.Context, batch []model.RequestBatchBody) ([]model.ResponseBatchBody, error) {
	wrap := func(err error) error { return fmt.Errorf("service batch: %w", err) }

	// валидируем вход и соберём порядок URL
	order := make([]string, 0, len(batch))
	seen := make(map[repository.URL]struct{}, len(batch))
	corr := make(map[repository.URL]string, len(batch))

	for _, b := range batch {
		if _, err := usvc.ParseURL(string(b.OriginalURL)); err != nil {
			return nil, wrap(err)
		}
		if _, ok := seen[b.OriginalURL]; ok {
			return nil, wrap(fmt.Errorf("double url in data %s", b.OriginalURL))
		}
		seen[b.OriginalURL] = struct{}{}
		order = append(order, string(b.OriginalURL))
		corr[b.OriginalURL] = b.CorrelationID
	}

	// resultMap: URL -> ShortURL (то, что в итоге вернём)
	result := make(map[repository.URL]repository.ShortURL, len(batch))

	// Транзакционный путь
	tx, ok := s.r.(repository.TxRunner)
	if !ok {
		err := createBatchFunc(ctx, order, result)(s.r)
		if err != nil {
			return nil, err
		}
	}

	if ok {
		err := tx.InTx(ctx, createBatchFunc(ctx, order, result))
		if err != nil {
			return nil, err
		}
	}
	// Формируем ответ
	out := make([]model.ResponseBatchBody, 0, len(batch))
	for _, u := range order {
		link, err := usvc.CreateURL(s.baseURL, result[repository.URL(u)])
		if err != nil {
			return nil, wrap(err)
		}
		out = append(out, model.ResponseBatchBody{
			CorrelationID: corr[repository.URL(u)],
			ShortURL:      link,
		})
	}
	return out, nil
}

// Batch func
func createBatchFunc(ctx context.Context, order []string, result map[repository.URL]repository.ShortURL) func(r repository.Repository) error {
	const retry = 6
	wrap := func(err error) error { return fmt.Errorf("service batch: %w", err) }

	batch := func(r repository.Repository) error {
		br, ok := r.(repository.BatchRepo)
		if !ok {
			return wrap(fmt.Errorf("repo in tx doesn't support batch methods"))
		}

		remaining := append([]string(nil), order...)

		// Добираем уже существующие
		existing, err := br.GetByURLs(ctx, remaining)
		if err != nil {
			return wrap(err)
		}
		for _, rec := range existing {
			result[rec.URL] = rec.ShortURL
		}
		remaining = urlsDiff(remaining, existing) // осталось только то, чего нет в БД
		if len(remaining) == 0 {
			return nil
		}

		// Пытаемся вставить оставшиеся, перегенерируя short только для оставшихся
		for attempt := 0; attempt < retry && len(remaining) > 0; attempt++ {
			args := make([]repository.ArgAddMany, 0, len(remaining))
			for _, u := range remaining {
				short := usvc.GenerateShort(6)
				args = append(args, repository.ArgAddMany{
					URL:      repository.URL(u),
					ShortURL: short,
				})
			}

			inserted, err := br.AddMany(ctx, args)
			if err != nil {
				return wrap(err)
			}

			for _, rec := range inserted {
				result[rec.URL] = rec.ShortURL
			}

			remaining = urlsDiff(remaining, inserted)
			if len(remaining) == 0 {
				break
			}

			// если кто-то из remaining не вставился потому что URL уже появился параллельно
			// доберём их Search'ем пачкой
			nowExist, err := br.GetByURLs(ctx, remaining)
			if err != nil {
				return wrap(err)
			}
			for _, rec := range nowExist {
				result[rec.URL] = rec.ShortURL
			}
			remaining = urlsDiff(remaining, nowExist)
		}

		if len(remaining) > 0 {
			return wrap(fmt.Errorf("can't generate unique short urls for %d items", len(remaining)))
		}
		return nil
	}
	return batch
}

// urlsDiff возвращает urls, которых нет среди records (по URL), сохраняя порядок.
func urlsDiff(urls []string, records []repository.Record) []string {
	existSet := make(map[string]struct{}, len(records))
	for _, r := range records {
		existSet[string(r.URL)] = struct{}{}
	}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		if _, ok := existSet[string(u)]; !ok {
			out = append(out, string(u))
		}
	}
	return out
}
