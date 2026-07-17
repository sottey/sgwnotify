package sgwnotify

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunTrimsConfigTokenAndNotifiesWithSkippedCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": " a.b.c ",
  "lookahead_minutes": 120,
  "open_url": "https://example.com",
  "http_timeout_seconds": 9
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	oldFetchFavoritesFunc := fetchFavoritesFunc
	oldNotifyEndingFavoritesFunc := notifyEndingFavoritesFunc
	oldNowFunc := nowFunc
	defer func() {
		fetchFavoritesFunc = oldFetchFavoritesFunc
		notifyEndingFavoritesFunc = oldNotifyEndingFavoritesFunc
		nowFunc = oldNowFunc
	}()

	var gotToken string
	fetchFavoritesFunc = func(token string, timeout time.Duration) ([]favorite, error) {
		gotToken = token
		if timeout != 9*time.Second {
			t.Fatalf("timeout got %s, want 9s", timeout)
		}
		return []favorite{
			{ItemID: 1, Title: "bad", EndTime: "bad-time"},
			{ItemID: 2, Title: "good", EndTime: "2026-06-04T12:15:00"},
		}, nil
	}

	var notified []favorite
	var skipped int
	var openURL string
	notifyEndingFavoritesFunc = func(favorites []favorite, skippedCount int, notificationOpenURL string) error {
		notified = favorites
		skipped = skippedCount
		openURL = notificationOpenURL
		return nil
	}
	nowFunc = func() time.Time {
		return time.Date(2026, 6, 4, 12, 0, 0, 0, time.Local)
	}

	var out bytes.Buffer
	if err := Run(Options{
		ConfigPath:              path,
		DefaultLookaheadMinutes: 120,
		DefaultOpenURL:          DefaultOpenURL,
		DefaultHTTPTimeout:      15 * time.Second,
		Verbose:                 true,
		Output:                  &out,
	}); err != nil {
		t.Fatal(err)
	}

	if gotToken != "a.b.c" {
		t.Fatalf("token got %q, want a.b.c", gotToken)
	}
	if len(notified) != 1 || notified[0].ItemID != 2 {
		t.Fatalf("unexpected notified favorites: %#v", notified)
	}
	if skipped != 1 {
		t.Fatalf("skipped got %d, want 1", skipped)
	}
	if openURL != "https://shopgoodwill.com/item/2" {
		t.Fatalf("openURL got %q, want item URL", openURL)
	}

	gotVerbose := out.String()
	for _, want := range []string{
		"Config path: " + path + "\n",
		"State path: " + filepath.Join(filepath.Dir(path), "notified.json") + "\n",
		"Lookahead minutes: 120\n",
		"HTTP timeout: 9s\n",
		"Favorites fetched: 2\n",
		"Matching favorites: 1\n",
		"Skipped bad end times: 1\n",
		"New matching favorites: 1\n",
	} {
		if !strings.Contains(gotVerbose, want) {
			t.Fatalf("verbose output missing %q:\n%s", want, gotVerbose)
		}
	}
}

func TestNotificationOpenURLUsesDefaultForMultipleFavorites(t *testing.T) {
	favorites := []favorite{
		{ItemID: 1},
		{ItemID: 2},
	}

	got := notificationOpenURL(favorites, "https://example.com")

	if got != "https://example.com" {
		t.Fatalf("got %q, want default URL", got)
	}
}

func TestRunSuppressesDuplicateNotification(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "a.b.c",
  "lookahead_minutes": 120
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	oldFetchFavoritesFunc := fetchFavoritesFunc
	oldNotifyEndingFavoritesFunc := notifyEndingFavoritesFunc
	oldNowFunc := nowFunc
	defer func() {
		fetchFavoritesFunc = oldFetchFavoritesFunc
		notifyEndingFavoritesFunc = oldNotifyEndingFavoritesFunc
		nowFunc = oldNowFunc
	}()

	fetchFavoritesFunc = func(token string, timeout time.Duration) ([]favorite, error) {
		return []favorite{
			{ItemID: 2, Title: "good", EndTime: "2026-06-04T12:15:00"},
		}, nil
	}
	nowFunc = func() time.Time {
		return time.Date(2026, 6, 4, 12, 0, 0, 0, time.Local)
	}

	notifications := 0
	notifyEndingFavoritesFunc = func(favorites []favorite, skippedCount int, notificationOpenURL string) error {
		notifications++
		return nil
	}

	opts := Options{
		ConfigPath:              path,
		DefaultLookaheadMinutes: 120,
		DefaultOpenURL:          DefaultOpenURL,
		DefaultHTTPTimeout:      15 * time.Second,
	}

	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}

	if notifications != 1 {
		t.Fatalf("notifications got %d, want 1", notifications)
	}
}

func TestRunSilentlySkipsEmptyFavorites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "a.b.c"
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	oldFetchFavoritesFunc := fetchFavoritesFunc
	oldNotifyEndingFavoritesFunc := notifyEndingFavoritesFunc
	defer func() {
		fetchFavoritesFunc = oldFetchFavoritesFunc
		notifyEndingFavoritesFunc = oldNotifyEndingFavoritesFunc
	}()

	fetchFavoritesFunc = func(token string, timeout time.Duration) ([]favorite, error) {
		return []favorite{}, nil
	}
	notifications := 0
	notifyEndingFavoritesFunc = func(favorites []favorite, skippedCount int, notificationOpenURL string) error {
		notifications++
		return nil
	}

	if err := Run(Options{
		ConfigPath:              path,
		DefaultLookaheadMinutes: 120,
		DefaultOpenURL:          DefaultOpenURL,
		DefaultHTTPTimeout:      15 * time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	if notifications != 0 {
		t.Fatalf("notifications got %d, want 0", notifications)
	}
}

func TestRunRejectsBadTokenShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "bad-token"
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{
		ConfigPath:              path,
		DefaultLookaheadMinutes: 120,
		DefaultOpenURL:          DefaultOpenURL,
		DefaultHTTPTimeout:      15 * time.Second,
	}); err == nil {
		t.Fatal("expected token validation error")
	}
}
