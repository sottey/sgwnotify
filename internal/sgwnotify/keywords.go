package sgwnotify

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type keywordListing struct {
	listing
	Keywords []string
	Start    time.Time
}

func newKeywordListings(keywords []string, listingsByKeyword map[string][]listing, since time.Time, notified map[string]string) []keywordListing {
	byID := make(map[int64]keywordListing)
	for _, keyword := range keywords {
		for _, item := range listingsByKeyword[keyword] {
			start, err := time.ParseInLocation(endTimeLayout, item.StartTime, time.Local)
			if err != nil || !start.After(since) || notified[keywordListingStateKey(item)] == item.StartTime {
				continue
			}
			found, ok := byID[item.ItemID]
			if !ok {
				found = keywordListing{listing: item, Start: start}
			}
			found.Keywords = append(found.Keywords, keyword)
			byID[item.ItemID] = found
		}
	}
	newItems := make([]keywordListing, 0, len(byID))
	for _, item := range byID {
		newItems = append(newItems, item)
	}
	sort.Slice(newItems, func(i, j int) bool { return newItems[i].Start.After(newItems[j].Start) })
	return newItems
}

func keywordListingStateKey(item listing) string {
	return fmt.Sprintf("%d", item.ItemID)
}

func markKeywordListingsNotified(state notificationState, listings []keywordListing) {
	if state.KeywordNotified == nil {
		state.KeywordNotified = map[string]string{}
	}
	for _, item := range listings {
		state.KeywordNotified[keywordListingStateKey(item.listing)] = item.StartTime
	}
}

func keywordListingsBody(listings []keywordListing) string {
	limit := len(listings)
	if limit > 2 {
		limit = 2
	}
	lines := make([]string, 0, limit+1)
	for _, item := range listings[:limit] {
		lines = append(lines, fmt.Sprintf("%s (%s)\n%s", truncateTitle(item.Title), strings.Join(item.Keywords, ", "), itemURL(item.ItemID)))
	}
	if len(listings) > limit {
		lines = append(lines, fmt.Sprintf("and %d more", len(listings)-limit))
	}
	return strings.Join(lines, "\n")
}
