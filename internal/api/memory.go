package api

import (
	"context"
	"sync"
)

// MemoryClient is an in-memory Client used by the sample commands and tests.
// Replace it with a real implementation.
type MemoryClient struct {
	mu    sync.Mutex
	Items []Item
}

func (c *MemoryClient) ListItems(ctx context.Context, p ListParams) ([]Item, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := []Item{} // empty, not nil: --json prints []
	for _, it := range c.Items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if p.State != "all" && it.State != p.State {
			continue
		}
		out = append(out, it)
		if p.Limit > 0 && len(out) == p.Limit {
			break
		}
	}
	return out, nil
}

func (c *MemoryClient) GetItem(ctx context.Context, id ItemID) (*Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.Items {
		if c.Items[i].ID == id {
			it := c.Items[i]
			return &it, nil
		}
	}
	return nil, NotFoundError{ID: id}
}

func (c *MemoryClient) DeleteItem(ctx context.Context, id ItemID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.Items {
		if c.Items[i].ID == id {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			return nil
		}
	}
	return NotFoundError{ID: id}
}
