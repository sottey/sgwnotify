package sgwnotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

const favoritesURL = "https://buyerapi.shopgoodwill.com/api/Favorite/GetAllFavoriteItemsByType?Type=all"
const itemListingURL = "https://buyerapi.shopgoodwill.com/api/Search/ItemListing"
const maxFavoritesAttempts = 3

var doFavoritesRequestFunc = doFavoritesRequest

type favorite struct {
	ItemID  int64  `json:"itemId"`
	Title   string `json:"title"`
	EndTime string `json:"endTime"`
	End     time.Time
}

type favoritesResponse struct {
	Message        string     `json:"message"`
	Status         bool       `json:"status"`
	IsUnauthorized bool       `json:"isUnauthorized"`
	Data           []favorite `json:"data"`
}

type listing struct {
	ItemID    int64  `json:"itemId"`
	Title     string `json:"title"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type itemListingResponse struct {
	IsUnauthorized bool `json:"isUnauthorized"`
	Data           struct {
		Items []listing `json:"items"`
	} `json:"data"`
}

type itemListingRequest struct {
	CatIDs                          string `json:"catIds"`
	CategoryID                      int    `json:"categoryId"`
	CategoryLevel                   int    `json:"categoryLevel"`
	CategoryLevelNo                 string `json:"categoryLevelNo"`
	ClosedAuctionDaysBack           string `json:"closedAuctionDaysBack"`
	ClosedAuctionEndingDate         string `json:"closedAuctionEndingDate"`
	HighPrice                       string `json:"highPrice"`
	IsFromHeaderMenuTab             bool   `json:"isFromHeaderMenuTab"`
	IsFromHomePage                  bool   `json:"isFromHomePage"`
	IsMultipleCategoryIDs           bool   `json:"isMultipleCategoryIds"`
	IsSize                          bool   `json:"isSize"`
	IsWeddingCategory               string `json:"isWeddingCatagory"`
	Layout                          string `json:"layout"`
	LowPrice                        string `json:"lowPrice"`
	Page                            string `json:"page"`
	PageSize                        string `json:"pageSize"`
	PartNumber                      string `json:"partNumber"`
	SavedSearchID                   int    `json:"savedSearchId"`
	SearchBuyNowOnly                string `json:"searchBuyNowOnly"`
	SearchCanadaShipping            string `json:"searchCanadaShipping"`
	SearchClosedAuctions            string `json:"searchClosedAuctions"`
	SearchDescriptions              string `json:"searchDescriptions"`
	SearchInternationalShippingOnly string `json:"searchInternationalShippingOnly"`
	SearchNoPickupOnly              string `json:"searchNoPickupOnly"`
	SearchOneCentShippingOnly       string `json:"searchOneCentShippingOnly"`
	SearchPickupOnly                string `json:"searchPickupOnly"`
	SearchText                      string `json:"searchText"`
	SearchUSOnlyShipping            string `json:"searchUSOnlyShipping"`
	SelectedCategoryIDs             string `json:"selectedCategoryIds"`
	SelectedGroup                   string `json:"selectedGroup"`
	SelectedSellerIDs               string `json:"selectedSellerIds"`
	SortColumn                      string `json:"sortColumn"`
	SortDescending                  string `json:"sortDescending"`
	UseBuyerPrefs                   string `json:"useBuyerPrefs"`
}

func fetchKeywordListings(token, keyword string, timeout time.Duration) ([]listing, error) {
	payload, err := json.Marshal(defaultItemListingRequest(keyword))
	if err != nil {
		return nil, err
	}
	client := http.Client{Timeout: timeout}
	for attempt := 1; attempt <= maxFavoritesAttempts; attempt++ {
		resp, err := doItemListingRequest(&client, token, payload)
		if err != nil {
			if isTransientRequestError(err) && attempt < maxFavoritesAttempts {
				continue
			}
			return nil, requestError(err, timeout)
		}
		if isTransientStatus(resp.StatusCode) && attempt < maxFavoritesAttempts {
			resp.Body.Close()
			continue
		}
		return parseItemListingResponse(resp)
	}
	return nil, fmt.Errorf("item listing request failed after %d attempts", maxFavoritesAttempts)
}

func defaultItemListingRequest(keyword string) itemListingRequest {
	return itemListingRequest{CatIDs: "", CategoryID: 0, CategoryLevel: 1, CategoryLevelNo: "1", ClosedAuctionDaysBack: "7", ClosedAuctionEndingDate: time.Now().Format("1/2/2006"), HighPrice: "999999", IsWeddingCategory: "false", LowPrice: "0", Page: "1", PageSize: "40", SearchCanadaShipping: "false", SearchClosedAuctions: "false", SearchDescriptions: "false", SearchInternationalShippingOnly: "false", SearchNoPickupOnly: "false", SearchOneCentShippingOnly: "false", SearchPickupOnly: "false", SearchText: keyword, SearchUSOnlyShipping: "true", SortColumn: "1", SortDescending: "true", UseBuyerPrefs: "true"}
}

func doItemListingRequest(client *http.Client, token string, payload []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, itemListingURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("could not create item listing request: %w", err)
	}
	setShopGoodwillHeaders(req, token)
	req.ContentLength = int64(len(payload))
	return client.Do(req)
}

func parseItemListingResponse(resp *http.Response) ([]listing, error) {
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, unauthorizedTokenError()
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("item listing request failed: HTTP %d", resp.StatusCode)
	}
	var parsed itemListingResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("could not parse item listing response: %w", err)
	}
	if parsed.IsUnauthorized {
		return nil, unauthorizedTokenError()
	}
	return parsed.Data.Items, nil
}

func fetchFavorites(token string, timeout time.Duration) ([]favorite, error) {
	client := http.Client{Timeout: timeout}
	var lastErr error

	for attempt := 1; attempt <= maxFavoritesAttempts; attempt++ {
		resp, err := doFavoritesRequestFunc(&client, token)
		if err != nil {
			lastErr = requestError(err, timeout)
			if isTransientRequestError(err) && attempt < maxFavoritesAttempts {
				continue
			}
			return nil, lastErr
		}

		if isTransientStatus(resp.StatusCode) && attempt < maxFavoritesAttempts {
			lastErr = fmt.Errorf("favorites request failed: HTTP %d", resp.StatusCode)
			resp.Body.Close()
			continue
		}

		return parseFavoritesResponse(resp)
	}

	return nil, lastErr
}

func doFavoritesRequest(client *http.Client, token string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, favoritesURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("could not create favorites request: %w", err)
	}

	setShopGoodwillHeaders(req, token)

	return client.Do(req)
}

func parseFavoritesResponse(resp *http.Response) ([]favorite, error) {
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, unauthorizedTokenError()
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("favorites request failed: HTTP %d", resp.StatusCode)
	}

	var parsed favoritesResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("could not parse favorites response: %w", err)
	}
	if parsed.IsUnauthorized {
		return nil, unauthorizedTokenError()
	}
	if !parsed.Status {
		return []favorite{}, nil
	}

	return parsed.Data, nil
}

func unauthorizedTokenError() error {
	return fmt.Errorf("ShopGoodwill API token is unauthorized or expired; update it with: sgwnotify config set-token \"$SHOPGOODWILL_TOKEN\"")
}

func requestError(err error, timeout time.Duration) error {
	if timeout > 0 && errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("favorites request timed out after %s", timeout)
	}
	return fmt.Errorf("favorites request failed: %w", err)
}

func isTransientRequestError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	return errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED)
}

func isTransientStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || (status >= 500 && status <= 599)
}

func setShopGoodwillHeaders(req *http.Request, token string) {
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("access-control-allow-credentials", "true")
	req.Header.Set("access-control-allow-origin", "*")
	req.Header.Set("authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://shopgoodwill.com")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("sec-ch-ua", `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	req.ContentLength = 0
}
