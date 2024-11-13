package rss

import (
	"io"
	"log"
	"net/http"
	"rsslab/utils"
	"slices"
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/tidwall/gjson"
	"golang.org/x/net/html"
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
	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed Feed
	feed.SiteURL = rule.URL
	if rule.Title == "" {
		rule.Title = "title"
	}
	s, err := cascadia.Compile(rule.Title)
	if err != nil {
		return nil, err
	}
	feed.Title = utils.CollapseWhitespace(extractText(s.MatchFirst(root)))

	var titleSel, urlSel, contentSel, datePublishedSel cascadia.Selector
	if rule.ItemTitle != "" {
		if titleSel, err = cascadia.Compile(rule.ItemTitle); err != nil {
			return nil, err
		}
	}
	if rule.ItemUrl != "" {
		if urlSel, err = cascadia.Compile(rule.ItemUrl); err != nil {
			return nil, err
		}
	}
	if rule.ItemContent != "" {
		if contentSel, err = cascadia.Compile(rule.ItemContent); err != nil {
			return nil, err
		}
	}
	if rule.ItemDatePublished != "" {
		if datePublishedSel, err = cascadia.Compile(rule.ItemDatePublished); err != nil {
			return nil, err
		}
	}

	s, err = cascadia.Compile(rule.Items)
	if err != nil {
		return nil, err
	}
	items := s.MatchAll(root)
	feed.Items = make([]Item, 0, len(items))
	for _, item := range items {
		var i Item

		title := item
		if titleSel != nil {
			title = titleSel.MatchFirst(item)
		}
		i.Title = utils.CollapseWhitespace(extractText(title))

		url := item
		if urlSel != nil {
			url = urlSel.MatchFirst(item)
		}
		if rule.ItemUrlAttr == "" {
			rule.ItemUrlAttr = "href"
		}
		for _, attr := range url.Attr {
			if attr.Key == rule.ItemUrlAttr {
				i.URL = utils.AbsoluteUrl(attr.Val, rule.URL)
				i.GUID = i.URL
				break
			}
		}

		content := item
		if contentSel != nil {
			content = contentSel.MatchFirst(item)
		}
		var b strings.Builder
		if err := html.Render(&b, content); err != nil {
			return nil, err
		}
		i.Content = strings.TrimSpace(utils.Sanitize(rule.URL, b.String()))

		date := item
		if datePublishedSel != nil {
			date = datePublishedSel.MatchFirst(item)
		}
		i.Date = utils.ParseDate(extractText(date))

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

	feed := Feed{SiteURL: rule.HomePageURL}
	if rule.Title != "" {
		feed.Title = j.Get(rule.Title).String()
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

func extractText(node *html.Node) string {
	if node == nil {
		return ""
	}
	var b strings.Builder
	for n := range node.Descendants() {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
	}
	return b.String()
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
