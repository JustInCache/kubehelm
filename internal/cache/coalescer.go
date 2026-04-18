package cache

import (
	"context"

	"golang.org/x/sync/singleflight"
)

type Coalescer struct {
	group singleflight.Group
}

func NewCoalescer() *Coalescer {
	return &Coalescer{}
}

func (c *Coalescer) Do(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error, bool) {
	res, err, shared := c.group.Do(key, func() (any, error) {
		return fn(ctx)
	})
	return res, err, shared
}
