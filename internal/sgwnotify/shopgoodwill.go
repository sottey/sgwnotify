package sgwnotify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	sgw "github.com/seanottey/sgw-api"
)

type favorite struct {
	ItemID  int64
	Title   string
	EndTime string
	End     time.Time
}

type listing = sgw.Listing

var newShopGoodwillClientFunc = newShopGoodwillClient

func fetchFavorites(token string, timeout time.Duration) ([]favorite, error) {
	items, err := newShopGoodwillClientFunc(token, timeout).Favorites(context.Background())
	if err != nil {
		return nil, notifierAPIError(err)
	}

	favorites := make([]favorite, 0, len(items))
	for _, item := range items {
		favorites = append(favorites, favorite{
			ItemID:  item.ItemID,
			Title:   item.Title,
			EndTime: item.EndTime,
		})
	}
	return favorites, nil
}

func fetchKeywordListings(token string, keyword string, timeout time.Duration) ([]listing, error) {
	result, err := newShopGoodwillClientFunc(token, timeout).Search(context.Background(), sgw.DefaultSearch(keyword))
	if err != nil {
		return nil, notifierAPIError(err)
	}
	return result.Items, nil
}

func newShopGoodwillClient(token string, timeout time.Duration) *sgw.Client {
	client := sgw.NewClient()
	client.Token = strings.TrimSpace(token)
	client.HTTPClient = &http.Client{Timeout: timeout}
	return client
}

func notifierAPIError(err error) error {
	if errors.Is(err, sgw.ErrUnauthorized) {
		return fmt.Errorf("ShopGoodwill API token is unauthorized or expired; update it with: sgwnotify config set-token \"$SHOPGOODWILL_TOKEN\": %w", err)
	}
	return err
}
