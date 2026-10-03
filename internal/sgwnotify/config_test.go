package sgwnotify

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveConfigTokenPreservesExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "old",
  "lookahead_minutes": 45,
  "open_url": "https://example.com",
  "http_timeout_seconds": 9
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	if err := SaveConfigToken(path, "new"); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BearerToken != "new" {
		t.Fatalf("BearerToken got %q, want %q", cfg.BearerToken, "new")
	}
	if cfg.LookaheadMinutes != 45 {
		t.Fatalf("LookaheadMinutes got %d, want 45", cfg.LookaheadMinutes)
	}
	if cfg.OpenURL != "https://example.com" {
		t.Fatalf("OpenURL got %q, want https://example.com", cfg.OpenURL)
	}
	if cfg.HTTPTimeoutSeconds != 9 {
		t.Fatalf("HTTPTimeoutSeconds got %d, want 9", cfg.HTTPTimeoutSeconds)
	}
}

func TestSaveConfigTokenTrimsWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	if err := SaveConfigToken(path, " token \n"); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BearerToken != "token" {
		t.Fatalf("BearerToken got %q, want token", cfg.BearerToken)
	}
}

func TestSaveConfigLookaheadMinutesPreservesExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "token",
  "lookahead_minutes": 45,
  "open_url": "https://example.com",
  "http_timeout_seconds": 9
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	if err := SaveConfigLookaheadMinutes(path, 90); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BearerToken != "token" {
		t.Fatalf("BearerToken got %q, want token", cfg.BearerToken)
	}
	if cfg.LookaheadMinutes != 90 {
		t.Fatalf("LookaheadMinutes got %d, want 90", cfg.LookaheadMinutes)
	}
	if cfg.OpenURL != "https://example.com" {
		t.Fatalf("OpenURL got %q, want https://example.com", cfg.OpenURL)
	}
	if cfg.HTTPTimeoutSeconds != 9 {
		t.Fatalf("HTTPTimeoutSeconds got %d, want 9", cfg.HTTPTimeoutSeconds)
	}
}

func TestSaveConfigHTTPTimeoutSecondsPreservesExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "token",
  "lookahead_minutes": 45,
  "open_url": "https://example.com",
  "http_timeout_seconds": 9
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	if err := SaveConfigHTTPTimeoutSeconds(path, 30); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BearerToken != "token" {
		t.Fatalf("BearerToken got %q, want token", cfg.BearerToken)
	}
	if cfg.LookaheadMinutes != 45 {
		t.Fatalf("LookaheadMinutes got %d, want 45", cfg.LookaheadMinutes)
	}
	if cfg.OpenURL != "https://example.com" {
		t.Fatalf("OpenURL got %q, want https://example.com", cfg.OpenURL)
	}
	if cfg.HTTPTimeoutSeconds != 30 {
		t.Fatalf("HTTPTimeoutSeconds got %d, want 30", cfg.HTTPTimeoutSeconds)
	}
}

func TestConfigOpenURLDefault(t *testing.T) {
	got, err := ConfigOpenURL(filepath.Join(t.TempDir(), "missing.json"), "https://default.example")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://default.example" {
		t.Fatalf("got %q, want default URL", got)
	}
}

func TestValidateConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "a.b.c",
  "lookahead_minutes": 45,
  "open_url": "https://example.com",
  "http_timeout_seconds": 9
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateConfig(path); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfigRejectsBadTokenShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "not-a-jwt"
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateConfig(path); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateConfigRejectsExplicitZeroLookahead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "a.b.c",
  "lookahead_minutes": 0
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateConfig(path); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestShowConfigRedactsTokenAndShowsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
  "bearer_token": "1234567890abcdef"
}
	`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := ShowConfig(path, 120, "https://example.com", 15*time.Second, &out); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		`"bearer_token": "12345678...cdef"`,
		`"lookahead_minutes": 120`,
		`"open_url": "https://example.com"`,
		`"http_timeout_seconds": 15`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "1234567890abcdef") {
		t.Fatalf("output contains unredacted token:\n%s", got)
	}
}

func TestSaveConfigKeywordsNormalizesAndRejectsDuplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfigKeywords(path, []string{" watch ", "Vintage"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(cfg.Keywords, ","), "watch,Vintage"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if err := SaveConfigKeywords(path, []string{"watch", "WATCH"}); err == nil {
		t.Fatal("expected duplicate keyword error")
	}
}
