package inmemory

import (
	"fmt"

	st "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type Storage struct {
	dataShort map[st.ShortURL]st.URL
	dataURL   map[st.URL]st.ShortURL
}

func NewStorage() *Storage {
	return &Storage{
		dataShort: make(map[st.ShortURL]st.URL),
		dataURL:   make(map[st.URL]st.ShortURL),
	}
}

func (s *Storage) Get(shortURL st.ShortURL) (st.URL, error) {
	if url, ok := s.dataShort[shortURL]; ok {
		return url, nil
	}
	return "", st.ErrNotFoundShortURL
}

func (s *Storage) Add(shortURL st.ShortURL, url st.URL) error {
	if _, ok := s.dataShort[shortURL]; ok {
		return fmt.Errorf("%w: %v", st.ErrAlreadyExists, shortURL)
	} else if _, ok := s.dataURL[url]; ok {
		return fmt.Errorf("%w: %v", st.ErrAlreadyExists, url)
	} else {
		s.dataShort[shortURL] = url
		s.dataURL[url] = shortURL
		return nil
	}
}

func (s *Storage) Search(url st.URL) (st.ShortURL, error) {
	value, ok := s.dataURL[url]
	if ok {
		return value, nil
	}
	return "", st.ErrNotFoundURL
}
