package handlers

import "context"

type Pinger interface {
	PingContext(ctx context.Context) error
}
