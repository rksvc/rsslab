package rss

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"rsslab/utils"
	"strings"
	"time"
)

var ErrUnknownFormat = errors.New("unknown feed format")

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

func Parse(r io.Reader, baseUrl string) (*Feed, error) {
	lookup := make([]byte, 2048)
	n, err := io.ReadFull(r, lookup)
	if err == io.ErrUnexpectedEOF {
		lookup = lookup[:n]
		r = bytes.NewReader(lookup)
	} else if err != nil {
		return nil, err
	} else {
		r = io.MultiReader(bytes.NewReader(lookup), r)
	}

	_, parse := sniff(utils.BytesToString(lookup))
	if parse == nil {
		return nil, ErrUnknownFormat
	}
	feed, err := parse(r)
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	siteUrl, err := url.Parse(feed.SiteURL)
	if err != nil {
		return nil, err
	}
	siteUrl = base.ResolveReference(siteUrl)
	feed.SiteURL = siteUrl.String()

	for i := range feed.Items {
		itemUrl, err := url.Parse(feed.Items[i].URL)
		if err != nil {
			return nil, err
		}
		feed.Items[i].URL = siteUrl.ResolveReference(itemUrl).String()
	}

	return feed, nil
}

func sniff(lookup string) (string, func(r io.Reader) (*Feed, error)) {
	lookup = strings.TrimSpace(lookup)
loop:
	for {
		if len(lookup) > 0 {
			switch lookup[0] {
			case 0xEF, 0xBB, 0xBF, 0xFE, 0xFF:
				lookup = lookup[1:]
			default:
				break loop
			}
		} else {
			return "", nil
		}
	}

	switch lookup[0] {
	case '<':
		decoder := utils.XMLDecoder(strings.NewReader(lookup))
		for {
			token, _ := decoder.Token()
			if token == nil {
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				switch el.Name.Local {
				case "rss":
					return "rss", ParseRSS
				case "feed":
					return "atom", ParseAtom
				}
			}
		}
	case '{':
		return "json", ParseJSON
	}
	return "", nil
}
