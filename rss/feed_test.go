package rss

import (
	"reflect"
	"strings"
	"testing"
)

func TestSniff(t *testing.T) {
	testcases := []struct {
		input string
		want  string
	}{
		{
			`<?xml version="1.0"?><rss version="2.0"><channel></channel></rss>`,
			"rss",
		},
		{
			`<?xml version="1.0"?><rss version="2.0"><channel></channel></rss>`,
			"rss",
		},
		{
			`<?xml version="1.0" encoding="utf-8"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`,
			"atom",
		},
		{
			`{}`,
			"json",
		},
		{
			`<!DOCTYPE html><html><head><title></title></head><body></body></html>`,
			"",
		},
	}
	for _, testcase := range testcases {
		want := testcase.want
		have, _ := sniff(testcase.input)
		if want != have {
			t.Fatalf("input: %s\nwant: %#v\nhave: %#v", testcase.input, want, have)
		}
	}
}

func TestParse(t *testing.T) {
	have, err := Parse(strings.NewReader(`
		<?xml version="1.0"?>
		<rss version="2.0">
		   <channel>
			  <title>
				 Title
			  </title>
			  <item>
				 <title>
				  Item 1
				 </title>
				 <description>
					<![CDATA[<div>content</div>]]>
				 </description>
			  </item>
		   </channel>
		</rss>
	`), "")
	if err != nil {
		t.Fatal(err)
	}
	want := &Feed{
		Title: "Title",
		Items: []Item{
			{
				Title:   "Item 1",
				Content: "<div>content</div>",
			},
		},
	}
	if !reflect.DeepEqual(want, have) {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}

func TestParseShortFeed(t *testing.T) {
	have, err := Parse(strings.NewReader(
		`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`,
	), "")
	if err != nil {
		t.Fatal(err)
	}
	want := &Feed{}
	if !reflect.DeepEqual(want, have) {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}

func TestParseFeedWithBOM(t *testing.T) {
	have, err := Parse(strings.NewReader(
		"\xEF\xBB\xBF"+`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`,
	), "")
	if err != nil {
		t.Fatal(err)
	}
	want := &Feed{}
	if !reflect.DeepEqual(want, have) {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}
