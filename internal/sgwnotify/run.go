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
	nowFunc                   = time.Now
)

type resolvedOptions struct {
	ConfigPath       string
	StatePath        string
	Token            string
	LookaheadMinutes int
	OpenURL          string
	HTTPTimeout      time.Duration
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

	return resolvedOptions{
		ConfigPath:       opts.ConfigPath,
		StatePath:        statePathForConfig(opts.ConfigPath),
		Token:            token,
		LookaheadMinutes: lookaheadMinutes,
		OpenURL:          openURL,
		HTTPTimeout:      httpTimeout,
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
	if len(favorites) < 1 {
		fmt.Println("No favorites found")
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
	if len(ending) == 0 {
		return saveNotificationState(resolved.StatePath, state)
	}

	openURL := notificationOpenURL(ending, resolved.OpenURL)
	if err := notifyEndingFavoritesFunc(ending, skipped, openURL); err != nil {
		return err
	}
	markFavoritesNotified(state, ending)
	return saveNotificationState(resolved.StatePath, state)
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
