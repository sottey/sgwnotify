package sgwnotify

import (
	"fmt"
	"sort"
	"time"
)

const endTimeLayout = "2006-01-02T15:04:05"

func endingSoon(favorites []favorite, now time.Time, lookahead time.Duration) ([]favorite, error) {
	ending, _, err := endingSoonWithSkipped(favorites, now, lookahead)
	return ending, err
}

func endingSoonWithSkipped(favorites []favorite, now time.Time, lookahead time.Duration) ([]favorite, int, error) {
	deadline := now.Add(lookahead)
	ending := make([]favorite, 0)
	skipped := 0

	for _, item := range favorites {
		end, err := time.ParseInLocation(endTimeLayout, item.EndTime, time.Local)
		if err != nil {
			skipped++
			continue
		}
		item.End = end

		if !end.Before(now) && !end.After(deadline) {
			ending = append(ending, item)
		}
	}

	sort.Slice(ending, func(i, j int) bool {
		return ending[i].End.Before(ending[j].End)
	})

	return ending, skipped, nil
}

func itemURL(itemID int64) string {
	return fmt.Sprintf("https://shopgoodwill.com/item/%d", itemID)
}
