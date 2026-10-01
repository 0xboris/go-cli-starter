// Package api is a placeholder for your domain client (HTTP API, database, files).
// Commands depend on the Client interface, never on a concrete implementation.
package api

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Item struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ItemFields lists the fields available to --json.
var ItemFields = []string{"id", "title", "state", "updatedAt"}

// ExportData returns only the requested fields (drives --json field selection).
func (i Item) ExportData(fields []string) map[string]any {
	m := make(map[string]any, len(fields))
	for _, f := range fields {
		switch f {
		case "id":
			m[f] = i.ID
		case "title":
			m[f] = i.Title
		case "state":
			m[f] = i.State
		case "updatedAt":
			m[f] = i.UpdatedAt
		}
	}
	return m
}

type ListParams struct {
	State string
	Limit int
}

type Client interface {
	ListItems(ctx context.Context, p ListParams) ([]Item, error)
	GetItem(ctx context.Context, id int) (*Item, error)
	DeleteItem(ctx context.Context, id int) error
}

// NotFoundError lets commands produce a specific message.
type NotFoundError struct{ ID int }

func (e NotFoundError) Error() string { return fmt.Sprintf("item %d not found", e.ID) }

// MemoryClient is an in-memory Client used by the sample commands and tests.
// Replace it with a real implementation.
type MemoryClient struct {
	mu    sync.Mutex
	Items []Item
}

func (c *MemoryClient) ListItems(ctx context.Context, p ListParams) ([]Item, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []Item
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

func (c *MemoryClient) GetItem(_ context.Context, id int) (*Item, error) {
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

func (c *MemoryClient) DeleteItem(_ context.Context, id int) error {
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
