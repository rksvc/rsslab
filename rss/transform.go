package rss

import (
	"io"
	"log"
	"net/http"
	"rsslab/utils"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
	URL               string            `json:"url"`
	HomePageURL       string            `json:"home_page_url"`
	Headers           map[string]string `json:"headers"`
	Title             string            `json:"title"`
	Items             string            `json:"items"`
	ItemTitle         string            `json:"item_title"`
	ItemUrl           string            `json:"item_url"`
	ItemUrlPrefix     string            `json:"item_url_prefix"`
	ItemContent       string            `json:"item_content"`
	ItemDatePublished string            `json:"item_date_published"`
}

func TransformHTML(rule *HTMLRule, client *http.Client) (*Feed, error) {
	resp, err := tryGet(rule.URL, nil, client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed Feed
	feed.SiteURL = rule.URL
	if rule.Title == "" {
		feed.Title = utils.CollapseWhitespace(doc.Find("title").First().Text())
	} else {
		feed.Title = rule.Title
	}

	items := doc.Find(rule.Items)
	feed.Items = make([]Item, 0, items.Length())
	for _, item := range items.EachIter() {
		var i Item

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
		i.GUID = i.URL

		content := item
		if rule.ItemContent != "" {
			content = item.Find(rule.ItemContent)
		}
		i.Content, err = content.Html()
		if err != nil {
			return nil, err
		}
		i.Content = strings.TrimSpace(utils.Sanitize(rule.URL, i.Content))

		date := item
		if rule.ItemDatePublished != "" {
			date = item.Find(rule.ItemDatePublished).First()
		}
		i.Date = utils.ParseDate(date.Text())

		feed.Items = append(feed.Items, i)
	}

	slices.SortStableFunc(feed.Items, cmpItem)
	return &feed, nil
}

func TransformJSON(rule *JSONRule, client *http.Client) (*Feed, error) {
	resp, err := tryGet(rule.URL, rule.Headers, client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	j := gjson.ParseBytes(b)

	var feed = Feed{
		Title:   rule.Title,
		SiteURL: rule.HomePageURL,
	}
	var items []gjson.Result
	if rule.Items == "" {
		items = j.Array()
	} else {
		items = j.Get(rule.Items).Array()
	}
	feed.Items = make([]Item, 0, len(items))
	for _, item := range items {
		var i Item

		if rule.ItemTitle != "" {
			i.Title = item.Get(rule.ItemTitle).String()
		}

		if rule.ItemUrl != "" {
			i.URL = item.Get(rule.ItemUrl).String()
			if rule.ItemUrlPrefix != "" {
				i.URL = rule.ItemUrlPrefix + i.URL
			}
			i.GUID = i.URL
		}

		if rule.ItemContent != "" {
			i.Content = item.Get(rule.ItemContent).String()
		}

		if rule.ItemDatePublished != "" {
			i.Date = utils.ParseDate(item.Get(rule.ItemDatePublished).String())
		}

		feed.Items = append(feed.Items, i)
	}

	slices.SortStableFunc(feed.Items, cmpItem)
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

func tryGet(url string, headers map[string]string, client *http.Client) (resp *http.Response, err error) {
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
		resp, err = client.Do(req)
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

func cmpItem(a, b Item) int {
	if b.Date == nil {
		if a.Date == nil {
			return 0
		}
		return -1
	} else if a.Date == nil {
		return 1
	}
	return b.Date.Compare(*a.Date)
}
