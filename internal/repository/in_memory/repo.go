package inmemory

import (
	"context"
	"fmt"
	"sync"

	handlers "github.com/IvanOplesnin/url-shortener/internal/handler"
	"github.com/IvanOplesnin/url-shortener/internal/logger"
	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type Repo struct {
	mu        sync.RWMutex
	dataShort map[repo.ShortURL]repo.URL
	dataURL   map[repo.URL]repo.ShortURL

	nextUserID int64
	users      map[int64]struct{}

	userShort map[int64]map[repo.ShortURL]repo.URL

	deleted map[repo.ShortURL]bool
}

func userID(ctx context.Context) (int64, error) {
	claims, ok := handlers.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return 0, fmt.Errorf("no claims in context")
	}
	return claims.UserID, nil
}

func NewRepo() *Repo {
	return &Repo{
		dataShort:  make(map[repo.ShortURL]repo.URL),
		dataURL:    make(map[repo.URL]repo.ShortURL),
		nextUserID: 0,
		users:      make(map[int64]struct{}),
		userShort:  make(map[int64]map[repo.ShortURL]repo.URL),
		deleted:    make(map[repo.ShortURL]bool),
	}
}

func (r *Repo) Get(ctx context.Context, shortURL repo.ShortURL) (repo.URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	uID, err := userID(ctx)
	if err != nil {
		return "", err
	}
	logger.Log.Debugf("r(inmemery).Get shortURL: %s; userID: %v", shortURL, uID)
	if value, ok := r.deleted[shortURL]; ok && value {
		return "", repo.ErrIsDeleted
	}
	if url, ok := r.dataShort[shortURL]; ok {
		if _, ok := r.userShort[uID]; ok {
			if _, ok := r.userShort[uID][shortURL]; !ok {
				return "", repo.ErrNotFoundShortURL
			}
			return url, nil
		}
	}
	return "", repo.ErrNotFoundShortURL
}

func (r *Repo) Add(ctx context.Context, shortURL repo.ShortURL, url repo.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.dataShort[shortURL]; ok {
		return fmt.Errorf("%w: %v", repo.ErrShortURLAlreadyExists, shortURL)
	}
	if _, ok := r.dataURL[url]; ok {
		return fmt.Errorf("%w: %v", repo.ErrAlreadyExists, url)
	}
	r.dataShort[shortURL] = url
	r.dataURL[url] = shortURL

	if claims, ok := handlers.ClaimsFromContext(ctx); ok && claims != nil && claims.UserID != 0 {
		logger.Log.Debugf("r(inmemory).Add clams.UserID: %v", claims.UserID)
		if _, ok := r.users[claims.UserID]; ok {
			if r.userShort[claims.UserID] == nil {
				r.userShort[claims.UserID] = make(map[repo.ShortURL]repo.URL)
			}
			r.userShort[claims.UserID][shortURL] = url
		}
	}

	return nil
}

func (r *Repo) Search(ctx context.Context, url repo.URL) (repo.ShortURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	uID, err := userID(ctx)
	if err != nil {
		return "", err
	}
	short, ok := r.dataURL[url]
	if !ok {
		return "", repo.ErrNotFoundURL
	}
	if value, ok := r.deleted[short]; ok && value {
		return "", repo.ErrIsDeleted
	}
	if _, ok := r.userShort[uID]; ok {
		if _, ok := r.userShort[uID][short]; !ok {
			return "", repo.ErrNotFoundShortURL
		}
	}
	return short, nil

}

func (r *Repo) Seed(records []repo.Record) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dataShort = make(map[repo.ShortURL]repo.URL, len(records))
	r.dataURL = make(map[repo.URL]repo.ShortURL, len(records))

	for _, rec := range records {
		r.dataShort[rec.ShortURL] = rec.URL
		r.dataURL[rec.URL] = rec.ShortURL
	}
}

func (r *Repo) Snapshot() []repo.Record {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]repo.Record, 0, len(r.dataShort))
	id := 0
	for short, url := range r.dataShort {
		out = append(out, repo.Record{ID: id, ShortURL: short, URL: url})
		id++
	}
	return out
}

func (r *Repo) Remove(short repo.ShortURL, url repo.URL) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.dataShort, short)
	delete(r.dataURL, url)
}

func (r *Repo) GetByURLs(_ context.Context, urls []string) ([]repo.Record, error) {
	if len(urls) == 0 {
		return []repo.Record{}, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]repo.Record, 0, len(urls))
	for _, u := range urls {
		url := repo.URL(u)
		if short, ok := r.dataURL[url]; ok {
			out = append(out, repo.Record{
				ShortURL: short,
				URL:      url,
			})
		}
	}
	return out, nil
}

func (r *Repo) AddMany(ctx context.Context, records []repo.ArgAddMany) ([]repo.Record, error) {
	if len(records) == 0 {
		return []repo.Record{}, nil
	}

	var userID int64
	if claims, ok := handlers.ClaimsFromContext(ctx); ok && claims != nil {
		userID = claims.UserID
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]repo.Record, 0, len(records))
	for _, rec := range records {
		if _, ok := r.dataShort[rec.ShortURL]; ok {
			continue
		}
		if _, ok := r.dataURL[rec.URL]; ok {
			return nil, fmt.Errorf("%w: %v", repo.ErrAlreadyExists, rec.URL)
		}

		r.dataShort[rec.ShortURL] = rec.URL
		r.dataURL[rec.URL] = rec.ShortURL

		if userID != 0 {
			if _, ok := r.users[userID]; ok {
				if r.userShort[userID] == nil {
					r.userShort[userID] = make(map[repo.ShortURL]repo.URL)
				}
				r.userShort[userID][rec.ShortURL] = rec.URL
			}
		}

		out = append(out, repo.Record{
			URL:      rec.URL,
			ShortURL: rec.ShortURL,
		})
	}
	return out, nil
}

func (r *Repo) AddUser(_ context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextUserID++
	id := r.nextUserID

	r.users[id] = struct{}{}
	r.userShort[id] = make(map[repo.ShortURL]repo.URL)

	return id, nil
}

func (r *Repo) GetUser(_ context.Context, id int64) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.users[id]; !ok {
		return 0, repo.ErrNotUserFound
	}
	return id, nil
}

// UserURLs возвращает ссылки ТЕКУЩЕГО пользователя.
// userID берём из ctx (его кладёт middleware).
func (r *Repo) UserURLs(ctx context.Context) ([]repo.Record, error) {
	claims, ok := handlers.ClaimsFromContext(ctx)
	if !ok || claims == nil || claims.UserID == 0 {
		return nil, repo.ErrNotUserFound
	}

	userID := claims.UserID

	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.userShort[userID]
	if !ok || len(m) == 0 {
		return []repo.Record{}, nil
	}

	out := make([]repo.Record, 0, len(m))
	for short, url := range m {
		out = append(out, repo.Record{
			ShortURL: short,
			URL:      url,
		})
	}
	return out, nil
}

func (r *Repo) DeletedBatch(ctx context.Context, userID int64, shortUrls []string) error {
	if userID == 0 {
		return nil
	}
	if len(shortUrls) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[userID]; !ok {
		return nil
	}

	userMap := r.userShort[userID]
	if userMap == nil {
		return nil
	}

	for _, s := range shortUrls {
		short := repo.ShortURL(s)

		if _, ok := userMap[short]; !ok {
			continue
		}

		r.deleted[short] = true
	}

	return nil
}

func (r *Repo) Undelete(ctx context.Context, shortURL repo.ShortURL) error {
	uID, err := userID(ctx)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[uID]; !ok {
		return nil
	}
	if _, ok := r.userShort[uID][shortURL]; !ok {
		return nil
	}
	delete(r.deleted, shortURL)
	return nil
}
