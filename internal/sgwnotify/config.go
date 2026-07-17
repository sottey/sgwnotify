package sgwnotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	BearerToken        string `json:"bearer_token"`
	LookaheadMinutes   int    `json:"lookahead_minutes,omitempty"`
	OpenURL            string `json:"open_url,omitempty"`
	HTTPTimeoutSeconds int    `json:"http_timeout_seconds,omitempty"`
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config{}, nil
		}
		return config{}, err
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, fmt.Errorf("invalid JSON: %w", err)
	}
	return cfg, nil
}

func SaveConfigToken(path string, token string) error {
	cfg, err := loadConfig(path)
	if err != nil {
		return err
	}
	cfg.BearerToken = strings.TrimSpace(token)

	return saveConfig(path, cfg)
}

func SaveConfigLookaheadMinutes(path string, minutes int) error {
	cfg, err := loadConfig(path)
	if err != nil {
		return err
	}
	cfg.LookaheadMinutes = minutes

	return saveConfig(path, cfg)
}

func SaveConfigHTTPTimeoutSeconds(path string, seconds int) error {
	cfg, err := loadConfig(path)
	if err != nil {
		return err
	}
	cfg.HTTPTimeoutSeconds = seconds

	return saveConfig(path, cfg)
}

func saveConfig(path string, cfg config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func ConfigOpenURL(path string, defaultOpenURL string) (string, error) {
	cfg, err := loadConfig(path)
	if err != nil {
		return "", err
	}
	if cfg.OpenURL == "" {
		return defaultOpenURL, nil
	}
	return cfg.OpenURL, nil
}

type displayConfig struct {
	BearerToken        string `json:"bearer_token"`
	LookaheadMinutes   int    `json:"lookahead_minutes"`
	OpenURL            string `json:"open_url"`
	HTTPTimeoutSeconds int    `json:"http_timeout_seconds"`
}

func ShowConfig(path string, defaultLookaheadMinutes int, defaultOpenURL string, defaultHTTPTimeout time.Duration, w io.Writer) error {
	cfg, err := loadConfig(path)
	if err != nil {
		return err
	}

	lookaheadMinutes := cfg.LookaheadMinutes
	if lookaheadMinutes <= 0 {
		lookaheadMinutes = defaultLookaheadMinutes
	}

	openURL := cfg.OpenURL
	if openURL == "" {
		openURL = defaultOpenURL
	}

	httpTimeoutSeconds := cfg.HTTPTimeoutSeconds
	if httpTimeoutSeconds <= 0 {
		httpTimeoutSeconds = int(defaultHTTPTimeout / time.Second)
	}

	out := displayConfig{
		BearerToken:        redactToken(cfg.BearerToken),
		LookaheadMinutes:   lookaheadMinutes,
		OpenURL:            openURL,
		HTTPTimeoutSeconds: httpTimeoutSeconds,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

func ValidateConfig(path string) error {
	cfg, err := loadConfig(path)
	if err != nil {
		return err
	}
	fields, err := configFields(path)
	if err != nil {
		return err
	}

	if err := validateToken(cfg.BearerToken); err != nil {
		return err
	}
	if fields["lookahead_minutes"] && cfg.LookaheadMinutes <= 0 {
		return fmt.Errorf("lookahead_minutes must be greater than 0")
	}
	if cfg.OpenURL != "" {
		parsed, err := url.ParseRequestURI(cfg.OpenURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("open_url must be a valid absolute URL")
		}
	}
	if fields["http_timeout_seconds"] && cfg.HTTPTimeoutSeconds <= 0 {
		return fmt.Errorf("http_timeout_seconds must be greater than 0")
	}

	return nil
}

func configFields(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]bool{}, nil
		}
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	fields := make(map[string]bool, len(raw))
	for name := range raw {
		fields[name] = true
	}
	return fields, nil
}

func validateToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("bearer token is missing; update it with: sgwnotify config set-token \"$SHOPGOODWILL_TOKEN\"")
	}
	if strings.ContainsAny(token, " \t\r\n") {
		return fmt.Errorf("bearer token must not contain whitespace")
	}
	if strings.Count(token, ".") != 2 {
		return fmt.Errorf("bearer token must look like a JWT with 3 dot-separated parts")
	}
	return nil
}

func redactToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 12 {
		return "..."
	}
	return token[:8] + "..." + token[len(token)-4:]
}
