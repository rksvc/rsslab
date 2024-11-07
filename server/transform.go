package server

import (
	"io"
	"log"
	"net/http"
	"rsslab/utils"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed/json"
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
	Title             string `json:"title"`
	Items             string `json:"items"`
	ItemTitle         string `json:"item_title"`
	ItemUrl           string `json:"item_url"`
	ItemUrlPrefix     string `json:"item_url_prefix"`
	ItemContent       string `json:"item_content"`
	ItemDatePublished string `json:"item_date_published"`
}

func (s *Server) TransformHTML(rule *HTMLRule) (*json.Feed, error) {
	resp, err := s.tryGet(rule.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed json.Feed
	feed.HomePageURL = rule.URL
	if rule.Title == "" {
		feed.Title = utils.CollapseWhitespace(doc.Find("title").First().Text())
	} else {
		feed.Title = rule.Title
	}

	for _, item := range doc.Find(rule.Items).EachIter() {
		var i json.Item

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
		if t, ok := utils.ParseDate(date.Text()); ok {
			if b, err := t.MarshalJSON(); err == nil {
				i.DatePublished = utils.BytesToString(b)
			}
		}

		feed.Items = append(feed.Items, &i)
	}

	return &feed, nil
}

func (s *Server) TransformJSON(rule *JSONRule) (*json.Feed, error) {
	resp, err := s.tryGet(rule.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	j := gjson.ParseBytes(b)

	var feed = json.Feed{
		Title:       rule.Title,
		HomePageURL: rule.HomePageURL,
	}
	var items []gjson.Result
	if rule.Items == "" {
		items = j.Array()
	} else {
		items = j.Get(rule.Items).Array()
	}
	feed.Items = make([]*json.Item, 0, len(items))
	for _, item := range items {
		var i json.Item

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
			if t, ok := utils.ParseDate(item.Get(rule.ItemDatePublished).String()); ok {
				if b, err := t.MarshalJSON(); err == nil {
					i.DatePublished = utils.BytesToString(b)
				}
			}
		}

		feed.Items = append(feed.Items, &i)
	}

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

func (s *Server) tryGet(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
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
