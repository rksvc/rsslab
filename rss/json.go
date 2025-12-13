package rss

import (
	"cmp"
	"encoding/json"
	"io"
)

type jsonFeed struct {
	Version string     `json:"version"`
	Title   string     `json:"title"`
	SiteURL string     `json:"home_page_url"`
	Items   []jsonItem `json:"items"`
}

type jsonItem struct {
	ID            string `json:"id"`
	URL           string `json:"url"`
	Title         string `json:"title"`
	Summary       string `json:"summary"`
	Text          string `json:"content_text"`
	HTML          string `json:"content_html"`
	DatePublished string `json:"date_published"`
	DateModified  string `json:"date_modified"`
}

func ParseJSON(r io.Reader) (*Feed, error) {
	var jsonFeed jsonFeed
	if err := json.NewDecoder(r).Decode(&jsonFeed); err != nil {
		return nil, err
	}

	feed := &Feed{
		Title:   jsonFeed.Title,
		SiteURL: jsonFeed.SiteURL,
	}
	for _, item := range jsonFeed.Items {
		feed.Items = append(feed.Items, Item{
			GUID:    item.ID,
			Date:    parseDate(cmp.Or(item.DatePublished, item.DateModified)),
			URL:     item.URL,
			Title:   item.Title,
			Content: cmp.Or(item.HTML, item.Text, item.Summary),
		})
	}
	return feed, nil
}
