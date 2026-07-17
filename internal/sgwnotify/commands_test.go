package sgwnotify

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckTokenPrintsSuccess(t *testing.T) {
	path := writeTestConfig(t)

	oldFetchFavoritesFunc := fetchFavoritesFunc
	defer func() {
		fetchFavoritesFunc = oldFetchFavoritesFunc
	}()

	fetchFavoritesFunc = func(token string, timeout time.Duration) ([]favorite, error) {
		return []favorite{
			{ItemID: 1, Title: "first", EndTime: "2026-06-04T12:15:00"},
			{ItemID: 2, Title: "second", EndTime: "2026-06-04T12:30:00"},
		}, nil
	}

	var out bytes.Buffer
	if err := CheckToken(testOptions(path), &out); err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), "Token is valid. 2 favorites fetched.\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestListEndingFavoritesPrintsMatches(t *testing.T) {
	path := writeTestConfig(t)

	oldFetchFavoritesFunc := fetchFavoritesFunc
	oldNowFunc := nowFunc
	defer func() {
		fetchFavoritesFunc = oldFetchFavoritesFunc
		nowFunc = oldNowFunc
	}()

	fetchFavoritesFunc = func(token string, timeout time.Duration) ([]favorite, error) {
		return []favorite{
			{ItemID: 1, Title: "bad", EndTime: "bad-time"},
			{ItemID: 2, Title: "good", EndTime: "2026-06-04T12:15:00"},
		}, nil
	}
	nowFunc = func() time.Time {
		return time.Date(2026, 6, 4, 12, 0, 0, 0, time.Local)
	}

	var out bytes.Buffer
	if err := ListEndingFavorites(testOptions(path), ListOptions{OutputMode: OutputPlain}, &out); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		"12:15 PM\tgood\thttps://shopgoodwill.com/item/2\n",
		"Skipped 1 favorite with bad end time.\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestListEndingFavoritesPrintsJSON(t *testing.T) {
	path := writeTestConfig(t)

	oldFetchFavoritesFunc := fetchFavoritesFunc
	oldNowFunc := nowFunc
	defer func() {
		fetchFavoritesFunc = oldFetchFavoritesFunc
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

	var out bytes.Buffer
	if err := ListEndingFavorites(testOptions(path), ListOptions{OutputMode: OutputJSON}, &out); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		`"favorites": [`,
		`"item_id": 2`,
		`"title": "good"`,
		`"url": "https://shopgoodwill.com/item/2"`,
		`"skipped_bad_end_times": 0`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func writeTestConfig(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "a.b.c",
  "lookahead_minutes": 120,
  "open_url": "https://example.com",
  "http_timeout_seconds": 9
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func testOptions(path string) Options {
	return Options{
		ConfigPath:              path,
		DefaultLookaheadMinutes: 120,
		DefaultOpenURL:          DefaultOpenURL,
		DefaultHTTPTimeout:      15 * time.Second,
	}
}
