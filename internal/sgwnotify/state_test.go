package sgwnotify

import (
	"path/filepath"
	"testing"
)

func TestNotificationStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notified.json")
	state := notificationState{
		Notified: map[string]string{
			"1": "2026-06-04T12:15:00",
		},
	}

	if err := saveNotificationState(path, state); err != nil {
		t.Fatal(err)
	}

	got, err := loadNotificationState(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Notified["1"] != "2026-06-04T12:15:00" {
		t.Fatalf("state got %#v", got)
	}
}

func TestUnnotifiedFavoritesUsesItemIDAndEndTime(t *testing.T) {
	state := notificationState{
		Notified: map[string]string{
			"1": "2026-06-04T12:15:00",
		},
	}
	favorites := []favorite{
		{ItemID: 1, EndTime: "2026-06-04T12:15:00"},
		{ItemID: 1, EndTime: "2026-06-04T12:30:00"},
	}

	got := unnotifiedFavorites(favorites, state)

	if len(got) != 1 || got[0].EndTime != "2026-06-04T12:30:00" {
		t.Fatalf("got %#v", got)
	}
}

func TestPruneNotificationStateRemovesMissingOrChangedFavorites(t *testing.T) {
	state := notificationState{
		Notified: map[string]string{
			"1": "2026-06-04T12:15:00",
			"2": "2026-06-04T12:30:00",
			"3": "2026-06-04T12:45:00",
		},
	}
	favorites := []favorite{
		{ItemID: 1, EndTime: "2026-06-04T12:15:00"},
		{ItemID: 2, EndTime: "2026-06-04T12:31:00"},
	}

	pruneNotificationState(state, favorites)

	if len(state.Notified) != 1 || state.Notified["1"] != "2026-06-04T12:15:00" {
		t.Fatalf("state got %#v", state)
	}
}
