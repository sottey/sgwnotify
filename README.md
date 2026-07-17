# sgwnotify

`sgwnotify` is a small macOS command-line notifier for ShopGoodwill favorites.

ShopGoodwill auctions are easy to watch and easy to forget. `sgwnotify` checks your saved favorites, finds auctions ending soon, and sends one concise desktop notification so you can decide whether to bid before the auction closes.

It is intentionally simple:

- Run it manually or from cron.
- Keep your ShopGoodwill bearer token in a local JSON config file.
- Configure the lookahead window, for example 2 hours.
- Get one notification listing the first two matching favorites.
- Avoid repeat notifications for the same favorite and end time.
- Click a single-item notification to open that item when `terminal-notifier` is installed.

## Why Use It?

ShopGoodwill has watchlists, but it does not always fit the way people actually monitor auctions. If you already know what you care about and just want a timely reminder before an auction ends, `sgwnotify` keeps that workflow lightweight.

Typical use:

1. Add items to your ShopGoodwill favorites.
2. Run `sgwnotify` every 15 minutes from cron.
3. Get notified when any favorite is inside your configured ending window.
4. Click through to the item or favorites page and decide what to do.

No daemon, database, browser automation, or background service is required.

## Features

- Fetches all ShopGoodwill favorites from the ShopGoodwill buyer API.
- Filters favorites by auction end time.
- Sends a single combined macOS notification.
- Shows up to two ending favorites, plus a count of additional matches.
- Supports config file values with CLI flag overrides.
- Provides config helper commands for token and lookahead time.
- Provides config validation, config path, and redacted config display commands.
- Provides token checking and plain text or JSON listing commands.
- Supports verbose run output for cron/debugging.
- Suppresses duplicate notifications with a small local state file.
- Supports clickable notifications through `terminal-notifier`.
- Falls back to AppleScript notifications if `terminal-notifier` is unavailable.
- Reports fatal app errors with the same notification path.
- Uses an HTTP timeout so cron jobs do not hang indefinitely.

## Requirements

- macOS
- Go
- A valid ShopGoodwill bearer token
- Optional: `terminal-notifier` for clickable notifications

Install `terminal-notifier` with Homebrew:

```sh
brew install terminal-notifier
```

## Build

From the repository root:

```sh
go build -o sgwnotify .
```

## Quick Start

Save your ShopGoodwill bearer token:

```sh
./sgwnotify config set-token "$SHOPGOODWILL_TOKEN"
```

Set the lookahead window:

```sh
./sgwnotify config set-lookahead-minutes 120
```

Set the HTTP timeout:

```sh
./sgwnotify config set-http-timeout-seconds 15
```

Validate config:

```sh
./sgwnotify config validate
```

Check token access:

```sh
./sgwnotify check-token
```

Test notifications:

```sh
./sgwnotify test-notification
```

Run the check:

```sh
./sgwnotify
```

## Configuration

Default config path:

```text
~/.config/sgwnotify/config.json
```

Example config:

```json
{
  "bearer_token": "...",
  "lookahead_minutes": 120,
  "open_url": "https://shopgoodwill.com/shopgoodwill/favorites",
  "http_timeout_seconds": 15
}
```

Duplicate notification state is stored next to the config file:

```text
~/.config/sgwnotify/notified.json
```

A favorite is considered already notified only when both the item ID and end time match.

| Field | Required | Default | Description |
|---|---:|---|---|
| `bearer_token` | yes | none | ShopGoodwill API bearer token |
| `lookahead_minutes` | no | `120` | Notify for auctions ending within this many minutes |
| `open_url` | no | `https://shopgoodwill.com/shopgoodwill/favorites` | URL opened by clickable notifications when multiple favorites match |
| `http_timeout_seconds` | no | `15` | Timeout for the favorites API request |

## Commands

Run the normal check:

```sh
./sgwnotify
```

Override token and lookahead from the command line:

```sh
./sgwnotify --token "$SHOPGOODWILL_TOKEN" --lookahead-minutes 90
```

Print run details:

```sh
./sgwnotify --verbose
```

Use a different config file:

```sh
./sgwnotify --config ./config.json
```

Save a bearer token:

```sh
./sgwnotify config set-token "$SHOPGOODWILL_TOKEN"
```

Save the lookahead window:

```sh
./sgwnotify config set-lookahead-minutes 120
```

Save the HTTP timeout:

```sh
./sgwnotify config set-http-timeout-seconds 15
```

Validate the config file:

```sh
./sgwnotify config validate
```

Show the active config with the token redacted:

```sh
./sgwnotify config show
```

Show the active config path:

```sh
./sgwnotify config path
```

Check whether the configured token can access favorites:

```sh
./sgwnotify check-token
```

List favorites ending within the lookahead window:

```sh
./sgwnotify list
```

List favorites as JSON:

```sh
./sgwnotify list --json
```

Send a test notification:

```sh
./sgwnotify test-notification
```

## Notifications

When `terminal-notifier` is installed, `sgwnotify` uses it and sets the notification click action to the specific item URL when exactly one favorite matches.

When multiple favorites match, the notification click action is:

```text
https://shopgoodwill.com/shopgoodwill/favorites
```

If `terminal-notifier` is not installed, `sgwnotify` falls back to:

```sh
osascript -e 'display notification ...'
```

The fallback notification still displays, but clicking the notification may open Script Editor instead of ShopGoodwill. Install `terminal-notifier` if you want the click action to open your browser.

## Cron

`sgwnotify` stores notified item IDs and end times so cron does not notify repeatedly for the same favorite while that favorite remains inside the lookahead window.

Example cron entry for every 15 minutes:

```cron
*/15 * * * * /Users/seanottey/projects/sgwnotify/sgwnotify
```

If cron cannot find `terminal-notifier`, provide a full `PATH`:

```cron
*/15 * * * * PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin /Users/seanottey/projects/sgwnotify/sgwnotify
```

You can edit cron with:

```sh
crontab -e
```

## Behavior

On each run, `sgwnotify`:

1. Loads config.
2. Applies CLI flag overrides.
3. Calls the ShopGoodwill favorites API.
4. Parses each favorite's auction end time.
5. Filters favorites ending within the lookahead window.
6. Sends one notification if any favorites match.
7. Exits.

If no favorites are ending soon, it exits silently.

If a favorite has a bad end time, that favorite is skipped. If matching favorites remain, the notification reports how many bad records were skipped.

If a fatal error occurs, such as a missing token, expired token, API failure, or bad JSON response, it sends an error notification and exits non-zero.

## Token Note

`sgwnotify` does not currently extract tokens from Chrome or any browser profile. Provide the bearer token in config or with `--token`.

### Getting a ShopGoodwill bearer token

1. Log in to [ShopGoodwill](https://shopgoodwill.com) in your browser.
2. Open any item page so the site makes its normal API requests.
3. Open your browser's Developer Tools, then select the **Network** tab.
4. Filter the requests to **Fetch/XHR**, and select a request to the ShopGoodwill buyer API (for example, a favorites request).
5. In that request's headers, find the `Authorization` request header. Its value is `Bearer <token>`.
6. Copy only the token after `Bearer `, then save it:

```sh
./sgwnotify config set-token "<token>"
```

Treat this token like a password: do not commit it, share it, or paste it into logs. It can expire, so repeat this process and update the saved token if `sgwnotify check-token` reports that it is unauthorized.

## Troubleshooting

Check that the binary works:

```sh
./sgwnotify --help
```

Check notifications:

```sh
./sgwnotify test-notification
```

Check that `terminal-notifier` is installed:

```sh
which terminal-notifier
```

If notifications work manually but not from cron, cron likely has a different `PATH`. Use the cron example above with an explicit `PATH`.

If the API reports unauthorized, update the token:

```sh
./sgwnotify config set-token "$SHOPGOODWILL_TOKEN"
```
