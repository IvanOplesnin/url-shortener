package persisted

import (
	"context"
	"fmt"

	"github.com/IvanOplesnin/url-shortener/internal/filestorage"
	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type Repo struct {
	base  repo.Repository
	s     repo.Seeder
	snap  repo.Snapshoter
	p     filestorage.Persister
	rb    repo.Rollback
	tx    repo.TxRunner
	batch repo.BatchRepo
}

func New(base repo.Repository, s repo.Seeder, snap repo.Snapshoter, p filestorage.Persister, rb repo.Rollback, tx repo.TxRunner, batch repo.BatchRepo) (*Repo, error) {
	records, err := p.Load()
	if err != nil {
		return nil, fmt.Errorf("persisted: load: %w", err)
	}
	if s != nil {
		s.Seed(records)
	}

	return &Repo{
		base:  base,
		s:     s,
		snap:  snap,
		p:     p,
		rb:    rb,
		tx:    tx,
		batch: batch,
	}, nil
}

func (r *Repo) Get(ctx context.Context, s repo.ShortURL) (repo.URL, error) {
	return r.base.Get(ctx, s)
}

func (r *Repo) Search(ctx context.Context, url repo.URL) (repo.ShortURL, error) {
	return r.base.Search(ctx, url)
}

func (r *Repo) Add(ctx context.Context, short repo.ShortURL, url repo.URL) error {
	if err := r.base.Add(ctx, short, url); err != nil {
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

func (r *Repo) InTx(ctx context.Context, fn func(r repo.Repository) error) error {
	if r.tx != nil {
		return r.tx.InTx(ctx, fn)
	}
	return fn(r)
}

func (r *Repo) GetByURLs(ctx context.Context, urls []string) ([]repo.Record, error) {
	if r.batch != nil {
		return r.batch.GetByURLs(ctx, urls)
	} else {
		return nil, fmt.Errorf("no implement batch in repo")
	}
}

func (r *Repo) AddMany(ctx context.Context, records []repo.ArgAddMany) ([]repo.Record, error) {
	if r.batch != nil {
		res, err := r.batch.AddMany(ctx, records)
		if err != nil {
			return nil, err
		}
		if r.snap != nil {
			if err := r.p.Save(r.snap.Snapshot()); err != nil {
				if r.rb != nil {
					for _, rec := range res {
						r.rb.Remove(rec.ShortURL, rec.URL)
					}
				}
				return nil, fmt.Errorf("persisted: save: %w", err)
			}
		}
		return res, nil
	} else {
		return nil, fmt.Errorf("no implement batch in repo")
	}
}
