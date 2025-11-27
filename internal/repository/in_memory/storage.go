package inmemory

import st "github.com/IvanOplesnin/url-shortener/internal/repository"

type Storage struct {
	Data map[st.ShortURL]st.URL
}

func NewStorage() *Storage {
	return &Storage{
		Data: make(map[st.ShortURL]st.URL),
	}
}

func (s *Storage) Get(shortURL st.ShortURL) (st.URL, error) {
	if url, ok := s.Data[shortURL]; ok {
		return url, nil
	} else {
		return "", st.ErrNotFoundShortURL
	}
}

func (s *Storage) Add(shortURL st.ShortURL, url st.URL) error {
	if _, ok := s.Data[shortURL]; ok {
		return st.ErrAlreadyExists
	} else {
		s.Data[shortURL] = url
		return nil
	}
}

func (s *Storage) Search(url st.URL) (st.ShortURL, error) {
	for k, v := range s.Data {
		if v == url {
			return k, nil
		}
	}
	return "", st.ErrNotFoundURL
}
