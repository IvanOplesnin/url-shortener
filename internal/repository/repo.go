package repository

import (
	"context"
	"errors"
)

type ShortURL string
type URL string

var ErrNotFoundShortURL = errors.New("not found shortURL")
var ErrNotFoundURL = errors.New("not found URL")
var ErrAlreadyExists = errors.New("already exists URL")
var ErrShortURLAlreadyExists = errors.New("already exist ShortURL")

type Repository interface {
	Add(ctx context.Context, key ShortURL, value URL) error
	Get(ctx context.Context, key ShortURL) (URL, error)
	Search(ctx context.Context, url URL) (ShortURL, error)
}

type BatchRepo interface {
	Repository
	GetByURLs(ctx context.Context, urls []string) ([]Record, error)
	AddMany(ctx context.Context, records []ArgAddMany) ([]Record, error)
}

type Seeder interface {
	Seed([]Record)
}

type Snapshoter interface {
	Snapshot() []Record
}

type Rollback interface {
	Remove(ShortURL, URL)
}

type Record struct {
	ID       int      `json:"id"`
	URL      URL      `json:"url"`
	ShortURL ShortURL `json:"short_url"`
}

type ArgAddMany struct {
	URL      URL      `json:"url"`
	ShortURL ShortURL `json:"short_url"`
}
