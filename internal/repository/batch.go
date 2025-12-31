package repository

import "context"

type TxRunner interface {
	InTx(ctx context.Context, fn func(r Repository) error) error
}
