package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ItemID identifies an item. Parse user input with ParseItemID once, at the edge,
// and pass the typed value around instead of strings or bare ints.
type ItemID int

// ParseItemID accepts "3" or "#3".
func ParseItemID(s string) (ItemID, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(s), "#"))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid item id: %q", s)
	}
	return ItemID(n), nil
}

func (id ItemID) String() string { return strconv.Itoa(int(id)) }

type Item struct {
	ID        ItemID    `json:"id"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ItemFields lists the fields available to --json. It is also where a real client
// would decide what to fetch (gh passes the requested fields to its query builder).
var ItemFields = []string{"id", "title", "state", "url", "updatedAt"}

// ExportData returns only the requested fields (drives --json field selection).
func (i Item) ExportData(fields []string) map[string]any {
	m := make(map[string]any, len(fields))
	for _, f := range fields {
		switch f {
		case "id":
			m[f] = int(i.ID)
		case "title":
			m[f] = i.Title
		case "state":
			m[f] = i.State
		case "url":
			m[f] = i.URL
		case "updatedAt":
			m[f] = i.UpdatedAt
		}
	}
	return m
}
