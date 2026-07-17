package sgwnotify

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const (
	OutputPlain = "plain"
	OutputJSON  = "json"
)

type ListOptions struct {
	OutputMode string
}

type listOutput struct {
	Favorites []listFavoriteOutput `json:"favorites"`
	Skipped   int                  `json:"skipped_bad_end_times"`
}

type listFavoriteOutput struct {
	ItemID  int64  `json:"item_id"`
	Title   string `json:"title"`
	EndTime string `json:"end_time"`
	URL     string `json:"url"`
}

func CheckToken(opts Options, w io.Writer) error {
	resolved, err := resolveOptions(opts)
	if err != nil {
		return err
	}

	favorites, err := fetchFavoritesFunc(resolved.Token, resolved.HTTPTimeout)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "Token is valid. %d favorite%s fetched.\n", len(favorites), plural(len(favorites)))
	return err
}

func ListEndingFavorites(opts Options, listOpts ListOptions, w io.Writer) error {
	resolved, err := resolveOptions(opts)
	if err != nil {
		return err
	}

	favorites, err := fetchFavoritesFunc(resolved.Token, resolved.HTTPTimeout)
	if err != nil {
		return err
	}

	ending, skipped, err := endingSoonWithSkipped(favorites, nowFunc(), time.Duration(resolved.LookaheadMinutes)*time.Minute)
	if err != nil {
		return err
	}

	if listOpts.OutputMode == "" {
		listOpts.OutputMode = OutputPlain
	}
	if listOpts.OutputMode == OutputJSON {
		return writeListJSON(ending, skipped, w)
	}
	if listOpts.OutputMode != OutputPlain {
		return fmt.Errorf("output mode must be plain or json")
	}

	if len(ending) == 0 {
		if _, err := fmt.Fprintln(w, "No favorites ending soon."); err != nil {
			return err
		}
	} else {
		for _, item := range ending {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", item.End.Format("3:04 PM"), item.Title, itemURL(item.ItemID)); err != nil {
				return err
			}
		}
	}

	if skipped > 0 {
		_, err = fmt.Fprintf(w, "Skipped %d favorite%s with bad end time.\n", skipped, plural(skipped))
		return err
	}

	return nil
}

func writeListJSON(favorites []favorite, skipped int, w io.Writer) error {
	out := listOutput{
		Favorites: make([]listFavoriteOutput, 0, len(favorites)),
		Skipped:   skipped,
	}
	for _, item := range favorites {
		out.Favorites = append(out.Favorites, listFavoriteOutput{
			ItemID:  item.ItemID,
			Title:   item.Title,
			EndTime: item.End.Format(time.RFC3339),
			URL:     itemURL(item.ItemID),
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}
