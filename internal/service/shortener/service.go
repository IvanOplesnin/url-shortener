package shortener

import (
	"errors"
	"fmt"

	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
	usvc "github.com/IvanOplesnin/url-shortener/internal/service/url"
)

type Service struct {
	r       repo.Repository
	baseURL string
}

type Result struct {
	Short  repo.ShortURL
	Link   string
	Exists bool
}

func New(r repo.Repository, baseURL string) *Service {
	return &Service{r: r, baseURL: baseURL}
}

func (s *Service) Shorten(u repo.URL) (Result, error) {
	if _, err := usvc.ParseURL(string(u)); err != nil {
		return Result{}, fmt.Errorf("invalid url: %w", err)
	}

	short, err := s.r.Search(u)
	if err == nil {
		link, err := usvc.CreateURL(s.baseURL, short)
		if err != nil {
			return Result{}, err
		}
		return Result{Short: short, Link: link, Exists: true}, nil
	}

	if !errors.Is(err, repo.ErrNotFoundURL) {
		return Result{}, err
	}

	short, err = usvc.AddRandomString(s.r, u)
	if err != nil {
		return Result{}, err
	}

	link, err := usvc.CreateURL(s.baseURL, short)

	if err != nil {
		return Result{}, err
	}
	return Result{Short: short, Link: link, Exists: false}, nil
}

func (s *Service) Resolve(short repo.ShortURL) (repo.URL, error) {
	return s.r.Get(short)
}
