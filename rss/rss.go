package rss

import (
	"cmp"
	"encoding/xml"
	"io"
	"path"
	"rsslab/utils"
	"strings"
)

type rssFeed struct {
	XMLName xml.Name  `xml:"rss"`
	Version string    `xml:"version,attr"`
	Title   string    `xml:"channel>title"`
	Link    string    `xml:"rss channel>link"`
	Items   []rssItem `xml:"channel>item"`
}

type rssItem struct {
	GUID        rssGuid        `xml:"guid"`
	Title       string         `xml:"rss title"`
	Link        string         `xml:"rss link"`
	Description string         `xml:"rss description"`
	PubDate     string         `xml:"pubDate"`
	Enclosures  []rssEnclosure `xml:"enclosure"`

	DublinCoreDate string `xml:"http://purl.org/dc/elements/1.1/ date"`
	ContentEncoded string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`

	OrigLink          string `xml:"http://rssnamespace.org/feedburner/ext/1.0 origLink"`
	OrigEnclosureLink string `xml:"http://rssnamespace.org/feedburner/ext/1.0 origEnclosureLink"`

	Torrent rssTorrent `xml:"torrent"`
}

type rssGuid struct {
	GUID        string `xml:",chardata"`
	IsPermaLink string `xml:"isPermaLink,attr"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type rssTorrent struct {
	PubDate string `xml:"pubDate"`
}

func ParseRSS(r io.Reader) (*Feed, error) {
	var rss rssFeed
	decoder := utils.XMLDecoder(r)
	decoder.DefaultSpace = "rss"
	if err := decoder.Decode(&rss); err != nil {
		return nil, err
	}

	feed := &Feed{
		Title:   strings.TrimSpace(rss.Title),
		SiteURL: strings.TrimSpace(rss.Link),
	}
	for _, item := range rss.Items {
		var podcastURL string
		for _, e := range item.Enclosures {
			if strings.HasPrefix(e.Type, "audio/") {
				podcastURL = e.URL
				if item.OrigEnclosureLink != "" && strings.Contains(podcastURL, path.Base(item.OrigEnclosureLink)) {
					podcastURL = item.OrigEnclosureLink
				}
				break
			}
		}

		var permalink string
		if item.GUID.IsPermaLink == "true" {
			permalink = item.GUID.GUID
		}

		feed.Items = append(feed.Items, Item{
			GUID:     strings.TrimSpace(item.GUID.GUID),
			Date:     parseDate(cmp.Or(item.DublinCoreDate, item.PubDate, item.Torrent.PubDate)),
			URL:      cmp.Or(item.OrigLink, item.Link, permalink),
			Title:    strings.TrimSpace(item.Title),
			Content:  cmp.Or(strings.TrimSpace(item.ContentEncoded), strings.TrimSpace(item.Description)),
			AudioURL: podcastURL,
		})
	}
	return feed, nil
}
