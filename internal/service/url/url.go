package url

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"time"

	st "github.com/IvanOplesnin/url-shortener/internal/repository"
)

func CreateURL(base string, id st.ShortURL) (string, error) {
	url, err := url.JoinPath(base, string(id))
	if err != nil {
		return "", fmt.Errorf("error createUrl: %w", err)
	}
	return url, nil
}

func ParseURL(urlRaw string) (st.URL, error) {
	if urlRaw == "" {
		return "", fmt.Errorf("empty body")
	}
	if _, err := url.Parse(urlRaw); err != nil {
		return "", fmt.Errorf("error parseUrl: %w", err)
	}
	return st.URL(urlRaw), nil
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

func AddRandomString(storage st.Repository, url st.URL) (st.ShortURL, error) {
	const retry = 6

	lettrs := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	for count := 0; count < retry; count++ {
		for i := range b {
			b[i] = lettrs[r.Intn(len(lettrs))]
		}
		err := storage.Add(st.ShortURL(b), url)
		if err == nil {
			return st.ShortURL(string(b)), nil
		}
		if errors.Is(err, st.ErrShortURLAlreadyExists) {
			continue
		}
		return "", fmt.Errorf("error addRandomString: %w", err)
	}

	return "", fmt.Errorf("error addRandomString: Can't generate random string")
}
