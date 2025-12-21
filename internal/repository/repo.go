package repository

import "errors"

type ShortURL string
type URL string

var ErrNotFoundShortURL = errors.New("not found shortURL")
var ErrNotFoundURL = errors.New("not found URL")
var ErrAlreadyExists = errors.New("already exists URL")
var ErrShortURLAlreadyExists = errors.New("already exist ShortURL")

type Repository interface {
	Add(key ShortURL, value URL) error
	Get(key ShortURL) (URL, error)
	Search(url URL) (ShortURL, error)
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
