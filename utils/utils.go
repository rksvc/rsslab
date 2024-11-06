package utils

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"unsafe"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
)

const USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

var Env = make(map[string]string)

func init() {
	for _, env := range os.Environ() {
		i := strings.IndexByte(env, '=')
		if i >= 0 {
			Env[env[:i]] = env[i+1:]
		} else {
			Env[env] = ""
		}
	}
}

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

func Errorf(messages []api.Message) error {
	errs := make([]error, len(messages))
	for i, m := range messages {
		var err error
		if m.Location == nil {
			err = errors.New(m.Text)
		} else {
			err = fmt.Errorf("%s:%d:%d %s", m.Location.File, m.Location.Line, m.Location.Column, m.Text)
		}
		errs[i] = err
	}
	return errors.Join(errs...)
}

func GetEncoding(resp *http.Response) encoding.Encoding {
	contentType := resp.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil
	}
	cs, ok := params["charset"]
	if !ok {
		return nil
	}
	e, _ := charset.Lookup(cs)
	return e
}
