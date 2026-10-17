package sgwnotify

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sgw "github.com/seanottey/sgw-api"
)

func TestFetchFavoritesUsesSGWAPIClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/Favorite/GetAllFavoriteItemsByType" || r.URL.Query().Get("Type") != "all" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization got %q", got)
		}
		_, _ = w.Write([]byte(`{"status":true,"data":[{"itemId":1,"title":"first","endTime":"2026-10-16T12:15:00"}]}`))
	}))
	defer server.Close()

	withTestShopGoodwillClient(t, server)
	favorites, err := fetchFavorites(" test-token ", 9*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(favorites) != 1 || favorites[0].ItemID != 1 || favorites[0].Title != "first" {
		t.Fatalf("unexpected favorites: %#v", favorites)
	}
}

func TestFetchKeywordListingsUsesSGWAPISearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/Search/ItemListing" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		_, _ = w.Write([]byte(`{"data":{"itemCount":1,"items":[{"itemId":2,"title":"watch","startTime":"2026-10-16T12:15:00","endTime":"2026-10-17T12:15:00"}]}}`))
	}))
	defer server.Close()

	withTestShopGoodwillClient(t, server)
	items, err := fetchKeywordListings("test-token", "watch", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ItemID != 2 || items[0].StartTime != "2026-10-16T12:15:00" {
		t.Fatalf("unexpected listings: %#v", items)
	}
}

func TestNotifierAPIErrorPreservesUnauthorizedCause(t *testing.T) {
	err := notifierAPIError(sgw.ErrUnauthorized)
	if !errors.Is(err, sgw.ErrUnauthorized) {
		t.Fatalf("error does not preserve unauthorized cause: %v", err)
	}
	if !strings.Contains(err.Error(), "sgwnotify config set-token") {
		t.Fatalf("error does not include token update instruction: %v", err)
	}
}

func TestNewShopGoodwillClientUsesNotifierCredentialsAndTimeout(t *testing.T) {
	client := newShopGoodwillClient(" token ", 9*time.Second)
	if client.Token != "token" {
		t.Fatalf("token got %q, want token", client.Token)
	}
	if client.HTTPClient == nil || client.HTTPClient.Timeout != 9*time.Second {
		t.Fatalf("HTTP timeout got %#v, want 9s", client.HTTPClient)
	}
}

func withTestShopGoodwillClient(t *testing.T, server *httptest.Server) {
	t.Helper()
	old := newShopGoodwillClientFunc
	newShopGoodwillClientFunc = func(token string, timeout time.Duration) *sgw.Client {
		return &sgw.Client{BaseURL: server.URL, HTTPClient: server.Client(), Token: strings.TrimSpace(token)}
	}
	t.Cleanup(func() { newShopGoodwillClientFunc = old })
}
