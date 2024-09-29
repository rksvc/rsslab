package rsshub

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"rsslab/utils"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type date time.Time

func (d date) MarshalJSON() ([]byte, error) {
	return time.Time(d).MarshalJSON()
}

func (d *date) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" {
		return nil
	}
	msec, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		*d = date(time.UnixMilli(msec))
		return nil
	}
	for _, layout := range []string{
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.DateTime,
		time.DateOnly,
		"Mon, 02 Jan 2006 15:04:05 MST",
		// https://stackoverflow.com/q/25953158
		"Mon, 02 Jan 2006 15:04:05 Z0700",
	} {
		var t time.Time
		t, err = time.Parse(`"`+layout+`"`, s)
		if err == nil {
			*d = date(t)
			return nil
		}
	}
	return err
}

type data struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Link        string     `json:"link"`
	Image       string     `json:"image"`
	Author      any        `json:"author"`
	Language    string     `json:"language"`
	FeedLink    string     `json:"feedLink"`
	Item        []dataItem `json:"item"`
}

type dataItem struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	PubDate         *date  `json:"pubDate"`
	Updated         *date  `json:"updated"`
	Link            string `json:"link"`
	Category        any    `json:"category"`
	Author          any    `json:"author"`
	Guid            string `json:"guid"`
	Id              string `json:"id"`
	Image           string `json:"image"`
	Banner          string `json:"banner"`
	Language        string `json:"language"`
	EnclosureUrl    string `json:"enclosure_url"`
	EnclosureType   string `json:"enclosure_type"`
	EnclosureTitle  string `json:"enclosure_title"`
	EnclosureLength int    `json:"enclosure_length"`
	ItunesDuration  any    `json:"itunes_duration"`
	Content         struct {
		Html string `json:"html"`
		Text string `json:"text"`
	} `json:"content"`
}

type jsonFeed struct {
	Version     string         `json:"version,omitempty"`
	Title       string         `json:"title,omitempty"`
	HomePageUrl string         `json:"home_page_url,omitempty"`
	FeedUrl     string         `json:"feed_url,omitempty"`
	Description string         `json:"description,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	Authors     []author       `json:"authors,omitempty"`
	Language    string         `json:"language,omitempty"`
	Items       []jsonFeedItem `json:"items"`
}

type jsonFeedItem struct {
	Id            string       `json:"id,omitempty"`
	Url           string       `json:"url,omitempty"`
	Title         string       `json:"title,omitempty"`
	ContentHtml   string       `json:"content_html,omitempty"`
	ContentText   string       `json:"content_text,omitempty"`
	Image         string       `json:"image,omitempty"`
	BannerImage   string       `json:"banner_image,omitempty"`
	DatePublished *date        `json:"date_published,omitempty"`
	DateModified  *date        `json:"date_modified,omitempty"`
	Authors       []author     `json:"authors,omitempty"`
	Tags          []string     `json:"tags,omitempty"`
	Language      string       `json:"language,omitempty"`
	Attachments   []attachment `json:"attachments,omitempty"`
}

type attachment struct {
	Url               string `json:"url,omitempty"`
	MimeType          string `json:"mime_type,omitempty"`
	Title             string `json:"title,omitempty"`
	SizeInBytes       int    `json:"size_in_bytes,omitempty"`
	DurationInSeconds any    `json:"duration_in_seconds,omitempty"`
}

type author struct {
	Name   string `json:"name,omitempty"`
	URL    string `json:"url,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

func toJSONFeed(v any) (feed *jsonFeed, err error) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	var data data
	err = json.Unmarshal(b, &data)
	if err != nil {
		return
	}

	var baseUrl *url.URL
	if data.Link != "" {
		baseUrl, err = url.Parse(data.Link)
		if err != nil {
			return
		}
		if baseUrl.Scheme == "" {
			baseUrl.Scheme = "http"
		}
	}
	resolveRelativeLink := func(s *goquery.Selection, attrName string) {
		if baseUrl == nil {
			return
		}
		if val, exists := s.Attr(attrName); exists {
			ref, err := url.Parse(val)
			if err != nil {
				return
			}
			s.SetAttr(attrName, baseUrl.ResolveReference(ref).String())
		}
	}

	data.Title = collapseWhitespace(data.Title)
	data.Description = collapseWhitespace(data.Description)
	var wg sync.WaitGroup
	wg.Add(len(data.Item))
	errs := make([]error, len(data.Item))
	for i := range data.Item {
		go func() {
			defer wg.Done()
			item := &data.Item[i]
			if item.Link != "" && baseUrl != nil {
				ref, err := url.Parse(item.Link)
				if err != nil {
					errs[i] = err
					return
				}
				item.Link = baseUrl.ResolveReference(ref).String()
			}

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(html.UnescapeString(item.Description)))
			if err != nil {
				errs[i] = err
				return
			}
			doc.Find("script").Remove()
			for _, s := range doc.Find("img").EachIter() {
				if _, exists := s.Attr("src"); !exists {
					src, exists := s.Attr("data-src")
					if !exists {
						src, exists = s.Attr("data-original")
					}
					if exists {
						s.SetAttr("src", src)
					}
				}
				for _, attrName := range []string{"onclick", "onerror", "onload"} {
					s.RemoveAttr(attrName)
				}
			}
			for _, s := range doc.Find("a, area").EachIter() {
				resolveRelativeLink(s, "href")
			}
			for _, s := range doc.Find("img, video, audio, source, iframe, embed, track").EachIter() {
				resolveRelativeLink(s, "src")
			}
			for _, s := range doc.Find("video[poster]").EachIter() {
				resolveRelativeLink(s, "poster")
			}
			for _, s := range doc.Find("img, iframe").EachIter() {
				s.SetAttr("referrerpolicy", "no-referrer")
			}
			var description strings.Builder
			body := doc.Find("body")
			for i := range body.Nodes {
				html, err := body.Eq(i).Html()
				if err != nil {
					errs[i] = err
					return
				}
				description.WriteString(html)
			}
			item.Description = strings.TrimSpace(description.String())

			item.Title = collapseWhitespace(html.UnescapeString(item.Title))
			item.Content.Html = strings.TrimSpace(item.Content.Html)
			item.Content.Text = strings.TrimSpace(item.Content.Text)
		}()
	}
	wg.Wait()
	if err = errors.Join(errs...); err != nil {
		return
	}
	slices.SortStableFunc(data.Item, func(a, b dataItem) int {
		if a.PubDate == nil {
			return 1
		} else if b.PubDate == nil {
			return -1
		}
		return time.Time(*b.PubDate).Compare(time.Time(*a.PubDate))
	})

	feed = new(jsonFeed)
	feed.Version = "https://jsonfeed.org/version/1.1"
	feed.Title = data.Title
	feed.HomePageUrl = data.Link
	feed.FeedUrl = data.FeedLink
	feed.Description = utils.FirstNonEmpty(data.Description, data.Title)
	feed.Icon = data.Image
	if data.Author != nil {
		feed.Authors = []author{{Name: fmt.Sprintf("%v", data.Author)}}
	}
	feed.Language = data.Language
	feed.Items = make([]jsonFeedItem, len(data.Item))
	for i := range data.Item {
		dst, src := &feed.Items[i], &data.Item[i]
		dst.Id = utils.FirstNonEmpty(src.Guid, src.Id, src.Link)
		dst.Url = src.Link
		dst.Title = src.Title
		dst.ContentHtml = utils.FirstNonEmpty(src.Content.Html, src.Description, src.Title)
		dst.ContentText = src.Content.Text
		dst.Image = src.Image
		dst.BannerImage = src.Banner
		dst.DatePublished = src.PubDate
		dst.DateModified = src.Updated
		dst.Authors = toAuthorArray(src.Author)
		dst.Tags = toStringArray(src.Category)
		dst.Language = src.Language
		if src.EnclosureUrl != "" {
			dst.Attachments = []attachment{{
				Url:               src.EnclosureUrl,
				MimeType:          src.EnclosureType,
				Title:             src.EnclosureTitle,
				SizeInBytes:       src.EnclosureLength,
				DurationInSeconds: src.ItunesDuration,
			}}
		}
	}
	return
}

func toStringArray(v any) (a []string) {
	switch v := v.(type) {
	case nil:
	case []any:
		for _, v := range v {
			if v != nil {
				a = append(a, fmt.Sprintf("%v", v))
			}
		}
	default:
		a = append(a, fmt.Sprintf("%v", v))
	}
	return
}

func toAuthorArray(v any) (a []author) {
	switch v := v.(type) {
	case nil:
	case []any:
		for _, v := range v {
			switch v := v.(type) {
			case string:
				a = append(a, author{Name: v})
			case map[string]any:
				var author author
				if name, ok := v["name"]; ok && name != nil {
					author.Name = fmt.Sprintf("%v", name)
				}
				if url, ok := v["url"]; ok && url != nil {
					author.URL = fmt.Sprintf("%v", url)
				}
				if avatar, ok := v["avatar"]; ok && avatar != nil {
					author.Avatar = fmt.Sprintf("%v", avatar)
				}
				a = append(a, author)
			}
		}
	default:
		a = append(a, author{Name: fmt.Sprintf("%v", v)})
	}
	return
}

var whitespaces = regexp.MustCompile(`\s+`)

func collapseWhitespace(s string) string {
	return whitespaces.ReplaceAllLiteralString(strings.TrimSpace(s), " ")
}
