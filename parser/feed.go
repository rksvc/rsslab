package parser

import (
	"io"
	"time"

	"github.com/nkanaev/yarr/src/parser"
)

type Feed struct {
	Title   string `json:"title,omitempty"`
	SiteURL string `json:"home_page_url,omitempty"`
	Items   []Item `json:"items,omitempty"`
}

type Item struct {
	GUID  string     `json:"id,omitempty"`
	Date  *time.Time `json:"date_published,omitempty"`
	URL   string     `json:"url,omitempty"`
	Title string     `json:"title,omitempty"`

	Content  string `json:"content_html,omitempty"`
	ImageURL string `json:"-"`
	AudioURL string `json:"-"`
}

func Parse(r io.Reader, baseUrl, fallbackEncoding string) (*Feed, error) {
	f, err := parser.ParseWithEncoding(r, fallbackEncoding)
	if err != nil {
		return nil, err
	}
	err = f.TranslateURLs(baseUrl)
	if err != nil {
		return nil, err
	}

	feed := Feed{
		Title:   f.Title,
		SiteURL: f.SiteURL,
		Items:   make([]Item, len(f.Items)),
	}
	for i, item := range f.Items {
		feed.Items[i] = Item{
			GUID:    item.GUID,
			URL:     item.URL,
			Title:   item.Title,
			Content: item.Content,
		}
		if !item.Date.IsZero() {
			feed.Items[i].Date = &item.Date
		}
		for _, media := range item.MediaLinks {
			switch media.Type {
			case "image":
				if feed.Items[i].ImageURL == "" {
					feed.Items[i].ImageURL = media.URL
				}
			case "audio":
				if feed.Items[i].AudioURL == "" {
					feed.Items[i].AudioURL = media.URL
				}
			}
		}
	}
	return &feed, nil
}
