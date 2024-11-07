package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unsafe"

	"golang.org/x/net/html"
)

const USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

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
