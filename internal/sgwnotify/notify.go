package sgwnotify

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	DefaultOpenURL = "https://shopgoodwill.com/shopgoodwill/favorites"
	maxTitleLength = 80
)

func notifyEndingFavorites(favorites []favorite, openURL string) error {
	title := fmt.Sprintf("%d favorite%s ending soon", len(favorites), plural(len(favorites)))
	return notify(title, endingFavoritesBody(favorites), openURL)
}

func notifyEndingFavoritesWithSkipped(favorites []favorite, skipped int, openURL string) error {
	title := fmt.Sprintf("%d favorite%s ending soon", len(favorites), plural(len(favorites)))
	body := endingFavoritesBody(favorites)
	if skipped > 0 {
		body = fmt.Sprintf("%s\nSkipped %d favorite%s with bad end time.", body, skipped, plural(skipped))
	}
	return notify(title, body, openURL)
}

func NotifyError(err error) {
	if err == nil {
		return
	}
	_ = notify("sgwnotify", err.Error(), DefaultOpenURL)
}

func TestNotification(openURL string) error {
	return notify("sgwnotify test", "Click Show to open ShopGoodwill favorites.", openURL)
}

func notify(title, body string, openURL string) error {
	if _, err := exec.LookPath("terminal-notifier"); err == nil {
		return notifyWithTerminalNotifier(title, body, openURL)
	}
	return notifyWithAppleScript(title, body)
}

func notifyWithTerminalNotifier(title, body string, openURL string) error {
	args := []string{
		"-title", title,
		"-message", body,
	}
	if openURL != "" {
		args = append(args, "-open", openURL)
	}

	cmd := exec.Command("terminal-notifier", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not send notification: %w", err)
	}
	return nil
}

func notifyWithAppleScript(title, body string) error {
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(
		"display notification %s with title %s",
		strconv.Quote(body),
		strconv.Quote(title),
	))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not send notification: %w", err)
	}
	return nil
}

func endingFavoritesBody(favorites []favorite) string {
	lines := make([]string, 0, 3)
	limit := len(favorites)
	if limit > 2 {
		limit = 2
	}

	for i := 0; i < limit; i++ {
		item := favorites[i]
		lines = append(lines, fmt.Sprintf("%s ends %s\n%s", truncateTitle(item.Title), item.End.Format("3:04 PM"), itemURL(item.ItemID)))
	}
	if len(favorites) > 2 {
		lines = append(lines, fmt.Sprintf("and %d more", len(favorites)-2))
	}

	return strings.Join(lines, "\n")
}

func truncateTitle(title string) string {
	runes := []rune(title)
	if len(runes) <= maxTitleLength {
		return title
	}
	return string(runes[:maxTitleLength-3]) + "..."
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func ExitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
