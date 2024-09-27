package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
)

var UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"
var Env = make(map[string]string)

func init() {
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			Env[kv[:i]] = kv[i+1:]
		}
	}
}

func FirstNonEmpty(vals ...string) string {
	for _, val := range vals {
		if val = strings.TrimSpace(val); val != "" {
			return val
		}
	}
	return ""
}

func ResponseError(resp *resty.Response) error {
	return fmt.Errorf(`%s "%s": %s`, resp.Request.Method, resp.Request.URL, resp.Status())
}

func IsErrorResponse(statusCode int) bool {
	return statusCode >= 400
}
