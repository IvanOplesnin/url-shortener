package filestorage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type Persister interface {
	Load() ([]repo.Record, error)
	Save([]repo.Record) error
}

type JSONStore struct {
	path string
}

func NewJSONStore(path string) *JSONStore { return &JSONStore{path: path} }

func (s *JSONStore) Save(records []repo.Record) error {
	const msg = "filestorage.JSONStore.Save"

	sort.Slice(records, func(i, j int) bool {
		return records[i].ShortURL < records[j].ShortURL
	})

	if dir := filepath.Dir(s.path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("%s: mkdir: %w", msg, err)
		}
	}

	tmp := s.path + ".tmp"
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

	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s: rename: %w", msg, err)
	}

	return nil
}

func (s *JSONStore) Load() ([]repo.Record, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	var records []repo.Record
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}
