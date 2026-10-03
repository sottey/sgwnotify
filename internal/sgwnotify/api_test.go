package sgwnotify

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSetShopGoodwillHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, favoritesURL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	setShopGoodwillHeaders(req, "test-token")

	wantHeaders := map[string]string{
		"accept":                           "application/json",
		"accept-language":                  "en-US,en;q=0.9",
		"access-control-allow-credentials": "true",
		"access-control-allow-origin":      "*",
		"authorization":                    "Bearer test-token",
		"content-type":                     "application/json",
		"origin":                           "https://shopgoodwill.com",
		"priority":                         "u=1, i",
		"sec-ch-ua":                        `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`,
		"sec-ch-ua-mobile":                 "?0",
		"sec-ch-ua-platform":               `"macOS"`,
		"sec-fetch-dest":                   "empty",
		"sec-fetch-mode":                   "cors",
		"sec-fetch-site":                   "same-site",
		"user-agent":                       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
	}

	for name, want := range wantHeaders {
		if got := req.Header.Get(name); got != want {
			t.Fatalf("%s: got %q, want %q", name, got, want)
		}
	}
	if req.ContentLength != 0 {
		t.Fatalf("ContentLength: got %d, want 0", req.ContentLength)
	}
}

func TestFetchFavoritesRetriesTransientStatus(t *testing.T) {
	oldDoFavoritesRequestFunc := doFavoritesRequestFunc
	defer func() {
		doFavoritesRequestFunc = oldDoFavoritesRequestFunc
	}()

	attempts := 0
	doFavoritesRequestFunc = func(client *http.Client, token string) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return response(http.StatusInternalServerError, `{"status":false}`), nil
		}
		return response(http.StatusOK, `{"status":true,"data":[{"itemId":1,"title":"item","endTime":"2026-06-04T12:15:00"}]}`), nil
	}

	favorites, err := fetchFavorites("a.b.c", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts got %d, want 2", attempts)
	}
	if len(favorites) != 1 || favorites[0].ItemID != 1 {
		t.Fatalf("unexpected favorites: %#v", favorites)
	}
}

func TestRequestErrorReportsTimeout(t *testing.T) {
	err := requestError(context.DeadlineExceeded, 15*time.Second)

	if got, want := err.Error(), "favorites request timed out after 15s"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseFavoritesResponseTreatsNoFavoritesAsEmpty(t *testing.T) {
	favorites, err := parseFavoritesResponse(response(http.StatusOK, `{"status":false,"message":"No favorites found"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(favorites) != 0 {
		t.Fatalf("got %#v, want no favorites", favorites)
	}
}

func TestDefaultItemListingRequestUsesNewestFirst(t *testing.T) {
	request := defaultItemListingRequest("watch")
	if request.SearchText != "watch" || request.SortColumn != "1" || request.SortDescending != "true" {
		t.Fatalf("unexpected search request: %#v", request)
	}
	if request.Page != "1" || request.PageSize != "40" || request.SearchUSOnlyShipping != "true" {
		t.Fatalf("unexpected paging or shipping defaults: %#v", request)
	}
}

func TestParseItemListingResponse(t *testing.T) {
	items, err := parseItemListingResponse(response(http.StatusOK, `{"data":{"items":[{"itemId":2,"title":"watch","startTime":"2026-10-03T12:00:00","endTime":"2026-10-04T12:00:00"}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ItemID != 2 || items[0].StartTime != "2026-10-03T12:00:00" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
