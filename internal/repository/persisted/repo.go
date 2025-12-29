package persisted

import (
	"fmt"

	"github.com/IvanOplesnin/url-shortener/internal/filestorage"
	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type Repo struct {
	base repo.Repository
	s    repo.Seeder
	snap repo.Snapshoter
	p    filestorage.Persister
	rb   repo.Rollback // maybe nil
}

func New(base repo.Repository, s repo.Seeder, snap repo.Snapshoter, p filestorage.Persister, rb repo.Rollback) (*Repo, error) {
	records, err := p.Load()
	if err != nil {
		return nil, fmt.Errorf("persisted: load: %w", err)
	}
	if s != nil {
		s.Seed(records)
	}

	return &Repo{
		base: base,
		s:    s,
		snap: snap,
		p:    p,
		rb:   rb,
	}, nil
}

func (r *Repo) Get(s repo.ShortURL) (repo.URL, error) {
	return r.base.Get(s)
}

func (r *Repo) Search(url repo.URL) (repo.ShortURL, error) {
	return r.base.Search(url)
}

func (r *Repo) Add(short repo.ShortURL, url repo.URL) error {
	if err := r.base.Add(short, url); err != nil {
		return err
	}

	if r.snap != nil {
		if err := r.p.Save(r.snap.Snapshot()); err != nil {
			if r.rb != nil {
				r.rb.Remove(short, url)
			}
			return fmt.Errorf("persisted: save: %w", err)
		}
	}

	return nil
}
