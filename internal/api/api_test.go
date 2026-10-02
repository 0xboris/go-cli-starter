package api

import (
	"context"
	"errors"
	"testing"
)

// Domain tests: no cobra, no IOStreams. This layer is testable on its own.

func sample() *MemoryClient {
	return &MemoryClient{Items: []Item{
		{ID: 3, Title: "c", State: "open"},
		{ID: 2, Title: "b", State: "closed"},
		{ID: 1, Title: "a", State: "open"},
	}}
}

func TestParseItemID(t *testing.T) {
	for in, want := range map[string]ItemID{"3": 3, "#12": 12, "7": 7} {
		got, err := ParseItemID(in)
		if err != nil || got != want {
			t.Errorf("ParseItemID(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "0", "-1", "#"} {
		if _, err := ParseItemID(in); err == nil {
			t.Errorf("ParseItemID(%q) succeeded, want error", in)
		}
	}
}

func TestListItems(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		p    ListParams
		want []ItemID
	}{
		{"open", ListParams{State: "open"}, []ItemID{3, 1}},
		{"all with limit", ListParams{State: "all", Limit: 2}, []ItemID{3, 2}},
		{"none", ListParams{State: "merged"}, []ItemID{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := sample().ListItems(ctx, tt.p)
			if err != nil {
				t.Fatal(err)
			}
			if items == nil {
				t.Fatal("ListItems returned nil; want an empty slice so --json prints []")
			}
			if len(items) != len(tt.want) {
				t.Fatalf("got %d items, want %d", len(items), len(tt.want))
			}
			for i, id := range tt.want {
				if items[i].ID != id {
					t.Errorf("item %d: id %d, want %d", i, items[i].ID, id)
				}
			}
		})
	}
}

func TestNotFoundIsTyped(t *testing.T) {
	c := sample()
	_, err := c.GetItem(context.Background(), 99)
	var nf NotFoundError
	if !errors.As(err, &nf) || nf.ID != 99 {
		t.Fatalf("GetItem(99) error = %v, want NotFoundError{99}", err)
	}
	if err := c.DeleteItem(context.Background(), 99); !errors.As(err, &nf) {
		t.Fatalf("DeleteItem(99) error = %v, want NotFoundError", err)
	}
}

func TestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := sample().ListItems(ctx, ListParams{State: "all"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListItems error = %v, want context.Canceled", err)
	}
	if err := sample().DeleteItem(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("DeleteItem error = %v, want context.Canceled", err)
	}
}

func TestExportDataSelectsFields(t *testing.T) {
	got := Item{ID: 3, Title: "c", State: "open", URL: "u"}.ExportData([]string{"id", "url"})
	if len(got) != 2 || got["id"] != 3 || got["url"] != "u" {
		t.Fatalf("ExportData = %v", got)
	}
}
