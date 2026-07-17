package sgwnotify

import (
	"testing"
	"time"
)

func TestEndingSoonFiltersAndSorts(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.Local)
	favorites := []favorite{
		{ItemID: 1, Title: "too late", EndTime: "2026-06-04T14:01:00"},
		{ItemID: 2, Title: "second", EndTime: "2026-06-04T13:30:00"},
		{ItemID: 3, Title: "first", EndTime: "2026-06-04T12:15:00"},
		{ItemID: 4, Title: "ended", EndTime: "2026-06-04T11:59:59"},
	}

	ending, err := endingSoon(favorites, now, 2*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if len(ending) != 2 {
		t.Fatalf("expected 2 ending favorites, got %d", len(ending))
	}
	if ending[0].ItemID != 3 || ending[1].ItemID != 2 {
		t.Fatalf("unexpected order: got %d, %d", ending[0].ItemID, ending[1].ItemID)
	}
}

func TestEndingSoonWithSkippedSkipsBadEndTimes(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.Local)
	favorites := []favorite{
		{ItemID: 1, Title: "bad", EndTime: "bad-time"},
		{ItemID: 2, Title: "good", EndTime: "2026-06-04T12:15:00"},
	}

	ending, skipped, err := endingSoonWithSkipped(favorites, now, 2*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped got %d, want 1", skipped)
	}
	if len(ending) != 1 || ending[0].ItemID != 2 {
		t.Fatalf("unexpected ending favorites: %#v", ending)
	}
}

func TestEndingFavoritesBodyShowsTwoAndMoreCount(t *testing.T) {
	favorites := []favorite{
		{ItemID: 1, Title: "first", End: time.Date(2026, 6, 4, 12, 15, 0, 0, time.Local)},
		{ItemID: 2, Title: "second", End: time.Date(2026, 6, 4, 12, 30, 0, 0, time.Local)},
		{ItemID: 3, Title: "third", End: time.Date(2026, 6, 4, 12, 45, 0, 0, time.Local)},
	}

	got := endingFavoritesBody(favorites)
	want := "first ends 12:15 PM\nhttps://shopgoodwill.com/item/1\nsecond ends 12:30 PM\nhttps://shopgoodwill.com/item/2\nand 1 more"
	if got != want {
		t.Fatalf("unexpected body:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTruncateTitle(t *testing.T) {
	title := "12345678901234567890123456789012345678901234567890123456789012345678901234567890extra"

	got := truncateTitle(title)
	want := "12345678901234567890123456789012345678901234567890123456789012345678901234567..."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
