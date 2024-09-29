package utils

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
)

var UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

func FirstNonEmpty(vals ...string) string {
	for _, val := range vals {
		if val = strings.TrimSpace(val); val != "" {
			return val
		}
	}
	return ""
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
