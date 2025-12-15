package inmemory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	l "github.com/IvanOplesnin/url-shortener/internal/logger"
	st "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type Storage struct {
	dataShort map[st.ShortURL]st.URL
	dataURL   map[st.URL]st.ShortURL
	filePath  string
}

type Record struct {
	URL      st.URL      `json:"url"`
	ShortURL st.ShortURL `json:"short_url"`
}

func NewStorage(filepath string) *Storage {
	st := &Storage{
		dataShort: make(map[st.ShortURL]st.URL),
		dataURL:   make(map[st.URL]st.ShortURL),
		filePath:  filepath,
	}

	err := st.Restore()
	if err != nil {
		l.Log.Fatalf("Error restore storage %s", err)
	}
	return st
}

func (s *Storage) Get(shortURL st.ShortURL) (st.URL, error) {
	if url, ok := s.dataShort[shortURL]; ok {
		return url, nil
	}
	return "", st.ErrNotFoundShortURL
}

func (s *Storage) Add(shortURL st.ShortURL, url st.URL) error {
	if _, ok := s.dataShort[shortURL]; ok {
		return fmt.Errorf("%w: %v", st.ErrShortURLAlreadyExists, shortURL)
	}
	if _, ok := s.dataURL[url]; ok {
		return fmt.Errorf("%w: %v", st.ErrAlreadyExists, url)
	}
	s.dataShort[shortURL] = url
	s.dataURL[url] = shortURL

	if err := s.addInFile(); err != nil {
		delete(s.dataShort, shortURL)
		delete(s.dataURL, url)
		return err
	}

	return nil
}

func (s *Storage) Search(url st.URL) (st.ShortURL, error) {
	value, ok := s.dataURL[url]
	if ok {
		return value, nil
	}
	return "", st.ErrNotFoundURL
}

func (s *Storage) addInFile() error {
	const msg = "storage.addInFile"

	records := make([]Record, 0, len(s.dataShort))
	for short, url := range s.dataShort {
		records = append(records, Record{
			ShortURL: short,
			URL:      url,
		})
	}

	if dir := filepath.Dir(s.filePath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("%s: mkdir: %w", msg, err)
		}
	}

	tmp := s.filePath + ".tmp"

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("%s: open tmp: %w", msg, err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(records); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("%s: encode: %w", msg, err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s: close: %w", msg, err)
	}

	if err := os.Rename(tmp, s.filePath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s: rename: %w", msg, err)
	}

	return nil
}

func (s *Storage) Restore() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	for _, r := range records {
		s.dataShort[r.ShortURL] = r.URL
		s.dataURL[r.URL] = r.ShortURL
	}
	return nil
}
