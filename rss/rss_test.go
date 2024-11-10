package rss

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRSSFeed(t *testing.T) {
	have, err := Parse(strings.NewReader(`
		<?xml version="1.0"?>
		<!DOCTYPE rss SYSTEM "http://my.netscape.com/publish/formats/rss-0.91.dtd">
		<rss version="0.91">
		<channel>
			<language>en</language>
			<description>???</description>
			<link>http://www.scripting.com/</link>
			<title>Scripting News</title>
			<item>
				<title>Title 1</title>
				<link>http://www.scripting.com/one/</link>
				<description>Description 1</description>
			</item>
			<item>
				<title>Title 2</title>
				<link>http://www.scripting.com/two/</link>
				<description>Description 2</description>
			</item>
		</channel>
		</rss>
	`), "")
	if err != nil {
		t.Fatal(err)
	}
	want := &Feed{
		Title:   "Scripting News",
		SiteURL: "http://www.scripting.com/",
		Items: []Item{
			{
				URL:     "http://www.scripting.com/one/",
				Title:   "Title 1",
				Content: "Description 1",
			},
			{
				URL:     "http://www.scripting.com/two/",
				Title:   "Title 2",
				Content: "Description 2",
			},
		},
	}
	if !reflect.DeepEqual(want, have) {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}

func TestRSSWithLotsOfSpaces(t *testing.T) {
	// https://pxlnv.com/: https://feedpress.me/pxlnv
	feed, err := Parse(strings.NewReader(strings.ReplaceAll(`
		<?xml version="1.0" encoding="UTF-8"?>
		<?xml-stylesheet type="text/xsl" media="screen" href="/~files/feed-premium.xsl"?>
		<lotsofspaces>
		<rss xmlns:content="http://purl.org/rss/1.0/modules/content/"
			 xmlns:wfw="http://wellformedweb.org/CommentAPI/"
			 xmlns:dc="http://purl.org/dc/elements/1.1/"
			 xmlns:atom="http://www.w3.org/2005/Atom"
			 xmlns:sy="http://purl.org/rss/1.0/modules/syndication/"
			 xmlns:slash="http://purl.org/rss/1.0/modules/slash/"
			 xmlns:feedpress="https://feed.press/xmlns"
			 xmlns:media="http://search.yahoo.com/mrss/"
			 version="2.0">
			<channel>
				<title>finally</title>
			</channel>
		</rss>
	`, "<lotsofspaces>", strings.Repeat(" ", 500))), "")
	if err != nil {
		t.Fatal(err)
	}
	have := feed.Title
	want := "finally"
	if have != want {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}

func TestRSSPodcast(t *testing.T) {
	feed, err := Parse(strings.NewReader(`
		<?xml version="1.0" encoding="UTF-8"?>
		<rss version="2.0">
			<channel>
				<item>
					<enclosure length="100500" type="audio/x-m4a" url="http://example.com/audio.ext"/>
				</item>
			</channel>
		</rss>
	`), "")
	if err != nil {
		t.Fatal(err)
	}
	have := feed.Items[0].AudioURL
	want := "http://example.com/audio.ext"
	if want != have {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}

func TestRSSOpusPodcast(t *testing.T) {
	feed, err := Parse(strings.NewReader(`
		<?xml version="1.0" encoding="UTF-8"?>
		<rss version="2.0">
			<channel>
				<item>
					<enclosure length="100500" type="audio/opus" url="http://example.com/audio.ext"/>
				</item>
			</channel>
		</rss>
	`), "")
	if err != nil {
		t.Fatal(err)
	}
	have := feed.Items[0].AudioURL
	want := "http://example.com/audio.ext"
	if want != have {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}

func TestRSSIsPermalink(t *testing.T) {
	feed, err := Parse(strings.NewReader(`
		<?xml version="1.0" encoding="UTF-8"?>
		<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
			<channel>
				<item>
					<guid isPermaLink="true">http://example.com/posts/1</guid>
				</item>
			</channel>
		</rss>
	`), "")
	if err != nil {
		t.Fatal(err)
	}
	have := feed.Items
	want := []Item{
		{
			GUID: "http://example.com/posts/1",
			URL:  "http://example.com/posts/1",
		},
	}
	if !reflect.DeepEqual(want, have) {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}

func TestRSSTorrentPubDate(t *testing.T) {
	feed, err := Parse(strings.NewReader(`
		<?xml version="1.0" encoding="UTF-8"?>
		<rss version="2.0">
			<channel>
				<item>
					<torrent>
						<link>https://example.com</link>
						<pubDate>2024-10-30T22:31:42.959</pubDate>
					</torrent>
				</item>
			</channel>
		</rss>
	`), "")
	if err != nil {
		t.Fatal(err)
	}
	have := *feed.Items[0].Date
	want := time.Date(2024, 10, 30, 22, 31, 42, 959_000_000, time.Local)
	if want != have {
		t.Fatalf("want: %#v\nhave: %#v", want, have)
	}
}
