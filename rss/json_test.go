package rss

import (
	"reflect"
	"strings"
	"testing"
)

func TestJSONFeed(t *testing.T) {
	have, err := Parse(strings.NewReader(`{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "2",
				"content_text": "This is a second item.",
				"url": "https://example.org/second-item"
			},
			{
				"id": "1",
				"content_html": "<p>Hello, world!</p>",
				"url": "https://example.org/initial-post"
			}
		]
	}`), "")
	if err != nil {
		t.Fatal(err)
	}
	want := &Feed{
		Title:   "My Example Feed",
		SiteURL: "https://example.org/",
		Items: []Item{
			{GUID: "2", Content: "This is a second item.", URL: "https://example.org/second-item"},
			{GUID: "1", Content: "<p>Hello, world!</p>", URL: "https://example.org/initial-post"},
		},
	}
	if !reflect.DeepEqual(want, have) {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}
