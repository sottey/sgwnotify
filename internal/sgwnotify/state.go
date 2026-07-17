package sgwnotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type notificationState struct {
	Notified map[string]string `json:"notified"`
}

func statePathForConfig(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "notified.json")
}

func loadNotificationState(path string) (notificationState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notificationState{Notified: map[string]string{}}, nil
		}
		return notificationState{}, err
	}

	var state notificationState
	if err := json.Unmarshal(data, &state); err != nil {
		return notificationState{}, fmt.Errorf("invalid notification state JSON: %w", err)
	}
	if state.Notified == nil {
		state.Notified = map[string]string{}
	}
	return state, nil
}

func saveNotificationState(path string, state notificationState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func pruneNotificationState(state notificationState, favorites []favorite) {
	current := make(map[string]string, len(favorites))
	for _, item := range favorites {
		current[favoriteStateKey(item)] = item.EndTime
	}

	for itemID, endTime := range state.Notified {
		if current[itemID] != endTime {
			delete(state.Notified, itemID)
		}
	}
}

func unnotifiedFavorites(favorites []favorite, state notificationState) []favorite {
	unnotified := make([]favorite, 0, len(favorites))
	for _, item := range favorites {
		if state.Notified[favoriteStateKey(item)] == item.EndTime {
			continue
		}
		unnotified = append(unnotified, item)
	}
	return unnotified
}

func markFavoritesNotified(state notificationState, favorites []favorite) {
	if state.Notified == nil {
		state.Notified = map[string]string{}
	}
	for _, item := range favorites {
		state.Notified[favoriteStateKey(item)] = item.EndTime
	}
}

func favoriteStateKey(item favorite) string {
	return strconv.FormatInt(item.ItemID, 10)
}
