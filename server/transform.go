package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"rsslab/utils"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	jsonfeed "github.com/mmcdole/gofeed/json"
	"github.com/tidwall/gjson"
)

type HTMLRule struct {
	URL               string `json:"url"`
	Title             string `json:"title"`
	Items             string `json:"items"`
	ItemTitle         string `json:"item_title"`
	ItemUrl           string `json:"item_url"`
	ItemUrlAttr       string `json:"item_url_attr"`
	ItemContent       string `json:"item_content"`
	ItemDatePublished string `json:"item_date_published"`
}

type JSONRule struct {
	URL               string `json:"url"`
	HomePageURL       string `json:"home_page_url"`
	Headers           string `json:"headers"`
	Title             string `json:"title"`
	Items             string `json:"items"`
	ItemTitle         string `json:"item_title"`
	ItemUrl           string `json:"item_url"`
	ItemUrlPrefix     string `json:"item_url_prefix"`
	ItemContent       string `json:"item_content"`
	ItemDatePublished string `json:"item_date_published"`
}

func (s *Server) TransformHTML(rule *HTMLRule) (*jsonfeed.Feed, error) {
	resp, err := s.tryGet(rule.URL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed jsonfeed.Feed
	feed.HomePageURL = rule.URL
	if rule.Title == "" {
		feed.Title = utils.CollapseWhitespace(doc.Find("title").First().Text())
	} else {
		feed.Title = rule.Title
	}

	for _, item := range doc.Find(rule.Items).EachIter() {
		var i jsonfeed.Item

		title := item
		if rule.ItemTitle != "" {
			title = item.Find(rule.ItemTitle)
		}
		i.Title = utils.ExtractText(title.Text())

		url := item
		if rule.ItemUrl != "" {
			url = item.Find(rule.ItemUrl)
		}
		if rule.ItemUrlAttr == "" {
			rule.ItemUrlAttr = "href"
		}
		i.URL = url.AttrOr(rule.ItemUrlAttr, "")
		i.URL = utils.AbsoluteUrl(i.URL, rule.URL)
		i.ID = i.URL

		content := item
		if rule.ItemContent != "" {
			content = item.Find(rule.ItemContent)
		}
		i.ContentHTML, err = content.Html()
		if err != nil {
			return nil, err
		}
		i.ContentHTML = strings.TrimSpace(utils.Sanitize(rule.URL, i.ContentHTML))

		date := item
		if rule.ItemDatePublished != "" {
			date = item.Find(rule.ItemDatePublished).First()
		}
		i.DatePublished = date.Text()

		feed.Items = append(feed.Items, &i)
	}

	sanitize(&feed)
	return &feed, nil
}

func (s *Server) TransformJSON(rule *JSONRule) (*jsonfeed.Feed, error) {
	var h map[string]string
	if rule.Headers != "" {
		if err := json.Unmarshal(utils.StringToBytes(rule.Headers), &h); err != nil {
			return nil, err
		}
	}
	resp, err := s.tryGet(rule.URL, h)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	j := gjson.ParseBytes(b)

	var feed = jsonfeed.Feed{
		Title:       rule.Title,
		HomePageURL: rule.HomePageURL,
	}
	var items []gjson.Result
	if rule.Items == "" {
		items = j.Array()
	} else {
		items = j.Get(rule.Items).Array()
	}
	feed.Items = make([]*jsonfeed.Item, 0, len(items))
	for _, item := range items {
		var i jsonfeed.Item

		if rule.ItemTitle != "" {
			i.Title = item.Get(rule.ItemTitle).String()
		}

		if rule.ItemUrl != "" {
			i.URL = item.Get(rule.ItemUrl).String()
			if rule.ItemUrlPrefix != "" {
				i.URL = rule.ItemUrlPrefix + i.URL
			}
			i.ID = i.URL
		}

		if rule.ItemContent != "" {
			i.ContentHTML = item.Get(rule.ItemContent).String()
		}

		if rule.ItemDatePublished != "" {
			i.DatePublished = item.Get(rule.ItemDatePublished).String()
		}

		feed.Items = append(feed.Items, &i)
	}

	sanitize(&feed)
	return &feed, nil
}

var retryStatusCodes = map[int]struct{}{
	http.StatusRequestTimeout:      {},
	http.StatusConflict:            {},
	http.StatusTooEarly:            {},
	http.StatusTooManyRequests:     {},
	http.StatusInternalServerError: {},
	http.StatusBadGateway:          {},
	http.StatusServiceUnavailable:  {},
	http.StatusGatewayTimeout:      {},
}

func (s *Server) tryGet(url string, headers map[string]string) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", utils.USER_AGENT)
	for key, val := range headers {
		req.Header.Set(key, val)
	}
	const maxTry = 3
	for attempt := 1; attempt <= maxTry; attempt++ {
		resp, err = s.client.Do(req)
		if err == nil {
			if !utils.IsErrorResponse(resp.StatusCode) {
				return
			}
			resp.Body.Close()
			err = utils.ResponseError(resp)
			if _, ok := retryStatusCodes[resp.StatusCode]; !ok {
				return
			}
		}
		if attempt < maxTry {
			log.Printf("%s, retry attempt %d", err, attempt)
		}
	}
	return
}

func sanitize(feed *jsonfeed.Feed) {
	date := make([]*time.Time, len(feed.Items))
	for i := range feed.Items {
		if d := &feed.Items[i].DatePublished; *d != "" {
			if t, ok := utils.ParseDate(*d); ok {
				if b, err := t.MarshalText(); err == nil {
					*d = utils.BytesToString(b)
					date[i] = &t
					continue
				}
			}
			*d = ""
		}
	}

	sort.SliceStable(feed.Items, func(i, j int) bool {
		if date[i] == nil {
			return true
		} else if date[j] == nil {
			return false
		}
		return date[j].Compare(*date[i]) < 0
	})
}
