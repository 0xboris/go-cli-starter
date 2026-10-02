package api

import (
	"context"
	"fmt"
)

// Client is the contract commands depend on. Keep it small; consumers that need
// less can declare a narrower interface of their own.
type Client interface {
	ListItems(ctx context.Context, p ListParams) ([]Item, error)
	GetItem(ctx context.Context, id ItemID) (*Item, error)
	DeleteItem(ctx context.Context, id ItemID) error
}

// ListParams holds an operation's inputs; add fields instead of positional params.
type ListParams struct {
	State string // "open", "closed" or "all"
	Limit int    // 0 means no limit
}

// NotFoundError lets commands produce a specific message.
type NotFoundError struct{ ID ItemID }

func (e NotFoundError) Error() string { return fmt.Sprintf("item %d not found", e.ID) }
