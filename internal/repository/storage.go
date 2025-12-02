package storage

import "errors"


type ShortURL string
type URL string


var ErrNotFoundShortURL = errors.New("not found shortURL")
var ErrNotFoundURL = errors.New("not found URL")
var ErrAlreadyExists = errors.New("already exists URL")
var ErrShortURLAlreadyExists = errors.New("already exist ShortURL")


type Storage interface {
	Add(key ShortURL, value URL) error
	Get(key ShortURL) (URL, error)
	Search(url URL) (ShortURL, error)
}

