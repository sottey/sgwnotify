package sgwnotify

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type Options struct {
	ConfigPath              string
	Token                   string
	TokenChanged            bool
	LookaheadMinutes        int
	LookaheadMinutesChanged bool
	DefaultLookaheadMinutes int
	DefaultOpenURL          string
	DefaultHTTPTimeout      time.Duration
	Verbose                 bool
	Output                  io.Writer
}

var (
	fetchFavoritesFunc        = fetchFavorites
	notifyEndingFavoritesFunc = notifyEndingFavoritesWithSkipped
	fetchKeywordListingsFunc  = fetchKeywordListings
	notifyKeywordListingsFunc = notifyKeywordListings
	nowFunc                   = time.Now
)

type resolvedOptions struct {
	ConfigPath       string
	StatePath        string
	Token            string
	LookaheadMinutes int
	OpenURL          string
	HTTPTimeout      time.Duration
	Keywords         []string
}

func resolveOptions(opts Options) (resolvedOptions, error) {
	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return resolvedOptions{}, fmt.Errorf("could not load config: %w", err)
	}

	token := strings.TrimSpace(cfg.BearerToken)
	if opts.TokenChanged {
		token = strings.TrimSpace(opts.Token)
	}
	if err := validateToken(token); err != nil {
		return resolvedOptions{}, err
	}

	lookaheadMinutes := cfg.LookaheadMinutes
	if opts.LookaheadMinutesChanged {
		lookaheadMinutes = opts.LookaheadMinutes
	}
	if lookaheadMinutes <= 0 {
		lookaheadMinutes = opts.DefaultLookaheadMinutes
	}

	openURL := cfg.OpenURL
	if openURL == "" {
		openURL = opts.DefaultOpenURL
	}

	httpTimeout := time.Duration(cfg.HTTPTimeoutSeconds) * time.Second
	if httpTimeout <= 0 {
		httpTimeout = opts.DefaultHTTPTimeout
	}
	keywords, err := normalizeKeywords(cfg.Keywords)
	if err != nil {
		return resolvedOptions{}, err
	}

	return resolvedOptions{
		ConfigPath:       opts.ConfigPath,
		StatePath:        statePathForConfig(opts.ConfigPath),
		Token:            token,
		LookaheadMinutes: lookaheadMinutes,
		OpenURL:          openURL,
		HTTPTimeout:      httpTimeout,
		Keywords:         keywords,
	}, nil
}

func Run(opts Options) error {
	resolved, err := resolveOptions(opts)
	if err != nil {
		return err
	}

	if opts.Verbose {
		writeVerbose(opts.Output, "Config path: %s\n", resolved.ConfigPath)
		writeVerbose(opts.Output, "State path: %s\n", resolved.StatePath)
		writeVerbose(opts.Output, "Lookahead minutes: %d\n", resolved.LookaheadMinutes)
		writeVerbose(opts.Output, "HTTP timeout: %s\n", resolved.HTTPTimeout)
	}

	favorites, err := fetchFavoritesFunc(resolved.Token, resolved.HTTPTimeout)
	if err != nil {
		return err
	}
	if len(favorites) == 0 && len(resolved.Keywords) == 0 {
		return nil
	}
	if opts.Verbose {
		writeVerbose(opts.Output, "Favorites fetched: %d\n", len(favorites))
	}

	ending, skipped, err := endingSoonWithSkipped(favorites, nowFunc(), time.Duration(resolved.LookaheadMinutes)*time.Minute)
	if err != nil {
		return err
	}
	if opts.Verbose {
		writeVerbose(opts.Output, "Matching favorites: %d\n", len(ending))
		writeVerbose(opts.Output, "Skipped bad end times: %d\n", skipped)
	}

	state, err := loadNotificationState(resolved.StatePath)
	if err != nil {
		return err
	}
	pruneNotificationState(state, favorites)
	ending = unnotifiedFavorites(ending, state)
	if opts.Verbose {
		writeVerbose(opts.Output, "New matching favorites: %d\n", len(ending))
	}
	if len(ending) > 0 {
		openURL := notificationOpenURL(ending, resolved.OpenURL)
		if err := notifyEndingFavoritesFunc(ending, skipped, openURL); err != nil {
			return err
		}
		markFavoritesNotified(state, ending)
	}
	if err := runKeywordSearches(resolved, &state, opts.Verbose, opts.Output); err != nil {
		return err
	}
	return saveNotificationState(resolved.StatePath, state)
}

func runKeywordSearches(resolved resolvedOptions, state *notificationState, verbose bool, output io.Writer) error {
	if len(resolved.Keywords) == 0 {
		return nil
	}
	listingsByKeyword := make(map[string][]listing, len(resolved.Keywords))
	for _, keyword := range resolved.Keywords {
		listings, err := fetchKeywordListingsFunc(resolved.Token, keyword, resolved.HTTPTimeout)
		if err != nil {
			return fmt.Errorf("keyword search for %q failed: %w", keyword, err)
		}
		listingsByKeyword[keyword] = listings
	}
	now := nowFunc()
	if state.KeywordLastCheckedAt == "" {
		for _, listings := range listingsByKeyword {
			for _, item := range listings {
				if _, err := time.ParseInLocation(endTimeLayout, item.StartTime, time.Local); err == nil {
					if state.KeywordNotified == nil {
						state.KeywordNotified = map[string]string{}
					}
					state.KeywordNotified[keywordListingStateKey(item)] = item.StartTime
				}
			}
		}
		state.KeywordLastCheckedAt = now.Format(time.RFC3339Nano)
		writeVerbose(output, "Keyword monitoring baseline established for %d keyword%s.\n", len(resolved.Keywords), plural(len(resolved.Keywords)))
		return nil
	}
	since, err := time.Parse(time.RFC3339Nano, state.KeywordLastCheckedAt)
	if err != nil {
		return fmt.Errorf("invalid keyword monitoring state timestamp: %w", err)
	}
	newListings := newKeywordListings(resolved.Keywords, listingsByKeyword, since, state.KeywordNotified)
	if verbose {
		writeVerbose(output, "New keyword listings: %d\n", len(newListings))
	}
	if len(newListings) > 0 {
		if err := notifyKeywordListingsFunc(newListings); err != nil {
			return err
		}
		markKeywordListingsNotified(*state, newListings)
	}
	state.KeywordLastCheckedAt = now.Format(time.RFC3339Nano)
	return nil
}

func notificationOpenURL(favorites []favorite, defaultOpenURL string) string {
	if len(favorites) == 1 {
		return itemURL(favorites[0].ItemID)
	}
	return defaultOpenURL
}

func writeVerbose(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format, args...)
}
