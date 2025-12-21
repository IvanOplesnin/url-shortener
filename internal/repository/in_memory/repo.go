package inmemory

import (
	"fmt"

	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type Repo struct {
	dataShort map[repo.ShortURL]repo.URL
	dataURL   map[repo.URL]repo.ShortURL
}

func NewRepo() *Repo {
	return &Repo{
		dataShort: make(map[repo.ShortURL]repo.URL),
		dataURL:   make(map[repo.URL]repo.ShortURL),
	}
}

func (r *Repo) Get(shortURL repo.ShortURL) (repo.URL, error) {
	if url, ok := r.dataShort[shortURL]; ok {
		return url, nil
	}
	return "", repo.ErrNotFoundShortURL
}

func (r *Repo) Add(shortURL repo.ShortURL, url repo.URL) error {
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

func (r *Repo) Search(url repo.URL) (repo.ShortURL, error) {
	value, ok := r.dataURL[url]
	if ok {
		return value, nil
	}
	return "", repo.ErrNotFoundURL
}

func (r *Repo) Seed(records []repo.Record) {
	r.dataShort = make(map[repo.ShortURL]repo.URL, len(records))
	r.dataURL = make(map[repo.URL]repo.ShortURL, len(records))

	for _, rec := range records {
		r.dataShort[rec.ShortURL] = rec.URL
		r.dataURL[rec.URL] = rec.ShortURL
	}
}

func (r *Repo) Snapshot() []repo.Record {
	out := make([]repo.Record, 0, len(r.dataShort))
	id := 0
	for short, url := range r.dataShort {
		out = append(out, repo.Record{ID: id, ShortURL: short, URL: url})
		id++
	}
	return out
}

func (r *Repo) Remove(short repo.ShortURL, url repo.URL) {
	delete(r.dataShort, short)
	delete(r.dataURL, url)
}
