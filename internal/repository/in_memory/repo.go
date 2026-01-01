package inmemory

import (
	"context"
	"fmt"
	"sync"

	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type Repo struct {
	mu        sync.RWMutex
	dataShort map[repo.ShortURL]repo.URL
	dataURL   map[repo.URL]repo.ShortURL
}

func NewRepo() *Repo {
	return &Repo{
		dataShort: make(map[repo.ShortURL]repo.URL),
		dataURL:   make(map[repo.URL]repo.ShortURL),
	}
}

func (r *Repo) Get(_ context.Context, shortURL repo.ShortURL) (repo.URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if url, ok := r.dataShort[shortURL]; ok {
		return url, nil
	}
	return "", repo.ErrNotFoundShortURL
}

func (r *Repo) Add(_ context.Context, shortURL repo.ShortURL, url repo.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.dataShort[shortURL]; ok {
		return fmt.Errorf("%w: %v", repo.ErrShortURLAlreadyExists, shortURL)
	}
	if _, ok := r.dataURL[url]; ok {
		return fmt.Errorf("%w: %v", repo.ErrAlreadyExists, url)
	}
	r.dataShort[shortURL] = url
	r.dataURL[url] = shortURL
	return nil
}

func (r *Repo) Search(_ context.Context, url repo.URL) (repo.ShortURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if short, ok := r.dataURL[url]; ok {
		return short, nil
	}
	return "", repo.ErrNotFoundURL
}

func (r *Repo) Seed(records []repo.Record) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dataShort = make(map[repo.ShortURL]repo.URL, len(records))
	r.dataURL = make(map[repo.URL]repo.ShortURL, len(records))

	for _, rec := range records {
		r.dataShort[rec.ShortURL] = rec.URL
		r.dataURL[rec.URL] = rec.ShortURL
	}
}

func (r *Repo) Snapshot() []repo.Record {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]repo.Record, 0, len(r.dataShort))
	id := 0
	for short, url := range r.dataShort {
		out = append(out, repo.Record{ID: id, ShortURL: short, URL: url})
		id++
	}
	return out
}

func (r *Repo) Remove(short repo.ShortURL, url repo.URL) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.dataShort, short)
	delete(r.dataURL, url)
}

func (r *Repo) GetByURLs(_ context.Context, urls []string) ([]repo.Record, error) {
	if len(urls) == 0 {
		return []repo.Record{}, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]repo.Record, 0, len(urls))
	for _, u := range urls {
		url := repo.URL(u)
		if short, ok := r.dataURL[url]; ok {
			out = append(out, repo.Record{
				ShortURL: short,
				URL:      url,
			})
		}
	}
	return out, nil
}

func (r *Repo) AddMany(_ context.Context, records []repo.ArgAddMany) ([]repo.Record, error) {
	if len(records) == 0 {
		return []repo.Record{}, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]repo.Record, 0, len(records))
	for _, rec := range records {
		if _, ok := r.dataShort[rec.ShortURL]; ok {
			continue
		}
		if _, ok := r.dataURL[rec.URL]; ok {
			return nil, fmt.Errorf("%w: %v", repo.ErrAlreadyExists, rec.URL)
		}

		r.dataShort[rec.ShortURL] = rec.URL
		r.dataURL[rec.URL] = rec.ShortURL

		out = append(out, repo.Record{
			URL:      rec.URL,
			ShortURL: rec.ShortURL,
		})
	}
	return out, nil
}
