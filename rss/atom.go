package rss

import (
	"encoding/xml"
	"io"
	"rsslab/utils"
	"strings"
)

type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	ID      string      `xml:"id"`
	Title   atomText    `xml:"title"`
	Links   atomLinks   `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID        string    `xml:"id"`
	Title     atomText  `xml:"title"`
	Summary   atomText  `xml:"summary"`
	Published string    `xml:"published"`
	Updated   string    `xml:"updated"`
	Links     atomLinks `xml:"link"`
	Content   atomText  `xml:"http://www.w3.org/2005/Atom content"`
	OrigLink  string    `xml:"http://rssnamespace.org/feedburner/ext/1.0 origLink"`
}

type atomText struct {
	Type string `xml:"type,attr"`
	Data string `xml:",chardata"`
	XML  string `xml:",innerxml"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type atomLinks []atomLink

func (a *atomText) Text() string {
	if a.Type == "html" {
		return utils.ExtractText(a.Data)
	} else if a.Type == "xhtml" {
		return utils.ExtractText(a.XML)
	}
	return strings.TrimSpace(a.Data)
}

func (a *atomText) String() string {
	data := a.Data
	if a.Type == "xhtml" {
		data = a.XML
	}
	return strings.TrimSpace(data)
}

func (links atomLinks) First(rel string) string {
	for _, link := range links {
		if link.Rel == rel {
			return link.Href
		}
	}
	return ""
}

func ParseAtom(r io.Reader) (*Feed, error) {
	var atom atomFeed
	if err := utils.XMLDecoder(r).Decode(&atom); err != nil {
		return nil, err
	}
	feed := &Feed{
		Title:   atom.Title.String(),
		SiteURL: utils.FirstNonEmpty(atom.Links.First("alternate"), atom.Links.First("")),
	}

	for _, item := range atom.Entries {
		var linkFromID, guidFromID string
		if utils.IsAPossibleLink(item.ID) {
			linkFromID = item.ID
			guidFromID = item.ID + "::" + item.Updated
		}
		link := utils.FirstNonEmpty(item.OrigLink, item.Links.First("alternate"), item.Links.First(""), linkFromID)
		feed.Items = append(feed.Items, Item{
			GUID:    utils.FirstNonEmpty(guidFromID, item.ID),
			Date:    utils.ParseDate(utils.FirstNonEmpty(item.Published, item.Updated)),
			URL:     link,
			Title:   item.Title.Text(),
			Content: utils.FirstNonEmpty(item.Content.String(), item.Summary.String()),
		})
	}
	return feed, nil
}
