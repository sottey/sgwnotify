package sgwnotify

import (
	"strings"
	"testing"
	"time"
)

func TestNewKeywordListingsFiltersDeduplicatesAndCombinesKeywords(t *testing.T) {
	since := time.Date(2026, 10, 3, 12, 0, 0, 0, time.Local)
	listings := map[string][]listing{
		"watch": {
			{ItemID: 1, Title: "old", StartTime: "2026-10-03T11:59:00"},
			{ItemID: 2, Title: "new", StartTime: "2026-10-03T12:01:00"},
		},
		"vintage": {
			{ItemID: 2, Title: "new", StartTime: "2026-10-03T12:01:00"},
			{ItemID: 3, Title: "already seen", StartTime: "2026-10-03T12:02:00"},
		},
	}

	got := newKeywordListings([]string{"watch", "vintage"}, listings, since, map[string]string{"3": "2026-10-03T12:02:00"})
	if len(got) != 1 || got[0].ItemID != 2 {
		t.Fatalf("got %#v", got)
	}
	if strings.Join(got[0].Keywords, ",") != "watch,vintage" {
		t.Fatalf("keywords got %#v", got[0].Keywords)
	}
}

func TestKeywordListingsBodyShowsKeywordsAndMoreCount(t *testing.T) {
	listings := []keywordListing{
		{listing: listing{ItemID: 1, Title: "first"}, Keywords: []string{"watch"}},
		{listing: listing{ItemID: 2, Title: "second"}, Keywords: []string{"vintage"}},
		{listing: listing{ItemID: 3, Title: "third"}, Keywords: []string{"clock"}},
	}
	got := keywordListingsBody(listings)
	for _, want := range []string{"first (watch)", "https://shopgoodwill.com/item/1", "second (vintage)", "and 1 more"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q: %s", want, got)
		}
	}
}
