package utils

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unsafe"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

func FirstNonEmpty(vals ...string) string {
	for _, val := range vals {
		if val = strings.TrimSpace(val); val != "" {
			return val
		}
	}
	return ""
}

func IsAPossibleLink(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func AbsoluteUrl(href, base string) string {
	hrefUrl, err := url.Parse(href)
	if err != nil {
		return ""
	}
	baseUrl, err := url.Parse(base)
	if err != nil {
		if hrefUrl.IsAbs() {
			return href
		}
		return ""
	}
	return baseUrl.ResolveReference(hrefUrl).String()
}

func UrlDomain(href string) string {
	if url, err := url.Parse(href); err == nil {
		return url.Host
	}
	return ""
}

var whitespaces = regexp.MustCompile(`\s+`)

func CollapseWhitespace(s string) string {
	return whitespaces.ReplaceAllLiteralString(strings.TrimSpace(s), " ")
}

func ExtractText(content string) string {
	var b strings.Builder
	tokenizer := html.NewTokenizer(strings.NewReader(content))
	for {
		token := tokenizer.Next()
		if token == html.ErrorToken {
			break
		}
		if token == html.TextToken {
			b.Write(tokenizer.Text())
		}
	}
	return CollapseWhitespace(b.String())
}

func ResponseError(resp *http.Response) error {
	return fmt.Errorf(`%s "%s": %s`, resp.Request.Method, resp.Request.URL, resp.Status)
}

func IsErrorResponse(statusCode int) bool {
	return statusCode >= 400
}

func XMLDecoder(r io.Reader) *xml.Decoder {
	d := xml.NewDecoder(r)
	d.Strict = false
	d.CharsetReader = charset.NewReaderLabel
	return d
}

func AddrOf[T any](val T) *T {
	return &val
}
