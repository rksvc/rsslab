package server

import "encoding/xml"

type OPML struct {
	XMLName  xml.Name  `xml:"opml"`
	Version  string    `xml:"version,attr,omitempty"`
	Title    string    `xml:"head>title,omitempty"`
	Outlines []Outline `xml:"body>outline"`
}

type Outline struct {
	Type     string    `xml:"type,attr,omitempty"`
	Title    string    `xml:"text,attr"`
	Title2   string    `xml:"title,attr,omitempty"`
	FeedUrl  string    `xml:"xmlUrl,attr,omitempty"`
	SiteUrl  string    `xml:"htmlUrl,attr,omitempty"`
	Outlines []Outline `xml:"outline,omitempty"`
}

func (o *Outline) isFolder() bool {
	return o.Type != "rss" && o.FeedUrl == ""
}

func (o *Outline) allFeeds() (outlines []Outline) {
	for _, o := range o.Outlines {
		if o.isFolder() {
			outlines = append(outlines, o.allFeeds()...)
		} else {
			outlines = append(outlines, o)
		}
	}
	return
}
