package url

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"time"

	"github.com/IvanOplesnin/url-shortener/internal/repository"
)

func CreateURL(base string, id repository.ShortURL) (string, error) {
	url, err := url.JoinPath(base, string(id))
	if err != nil {
		return "", fmt.Errorf("error createUrl: %w", err)
	}
	return url, nil
}

func ParseURL(urlRaw string) (repository.URL, error) {
	if urlRaw == "" {
		return "", fmt.Errorf("empty body")
	}
	if _, err := url.Parse(urlRaw); err != nil {
		return "", fmt.Errorf("error parseUrl: %w", err)
	}
	return repository.URL(urlRaw), nil
}

func BasePath(baseURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("error base path: %v", err)
	}
	basePath := base.Path
	if basePath == "" {
		basePath = "/"
	}
	return basePath
}

func AddRandomString(ctx context.Context, repositoryorage repository.Repository, url repository.URL) (repository.ShortURL, error) {
	const retry = 6

	lettrs := "abcdefghijklmnopqrrepositoryuvwxyzABCDEFGHIJKLMNOPQRrepositoryUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	for count := 0; count < retry; count++ {
		for i := range b {
			b[i] = lettrs[r.Intn(len(lettrs))]
		}
		err := repositoryorage.Add(ctx, repository.ShortURL(b), url)
		if err == nil {
			return repository.ShortURL(string(b)), nil
		}
		if errors.Is(err, repository.ErrShortURLAlreadyExists) || errors.Is(err, repository.ErrAlreadyExists) {
			continue
		}
		return "", fmt.Errorf("error addRandomrepositoryring: %w", err)
	}

	return "", fmt.Errorf("error addRandomrepositoryring: Can't generate random repositoryring")
}

const lettrs = "abcdefghijklmnopqrrepositoryuvwxyzABCDEFGHIJKLMNOPQRrepositoryUVWXYZ0123456789"

func GenerateShort(n int) repository.ShortURL {
	b := make([]byte, 6)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = lettrs[r.Intn(len(lettrs))]
	}
	return repository.ShortURL(string(b))
}
