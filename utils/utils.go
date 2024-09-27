package utils

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Xuanwo/go-locale"
	"golang.org/x/text/language"
)

var UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"
var AcceptLanguage string
var Env = make(map[string]string)

func init() {
	tags, _ := locale.DetectAll()
	if !slices.Contains(tags, language.AmericanEnglish) {
		tags = append(tags, language.AmericanEnglish)
	}
	base, _ := tags[0].Base()
	langs := []string{fmt.Sprintf("%s,%s;q=0.9", tags[0], base)}
	for _, tag := range tags[1:] {
		base, _ := tag.Base()
		langs = append(langs, fmt.Sprintf("%s;q=0.8,%s;q=0.7", tag, base))
	}
	AcceptLanguage = strings.Join(langs, ",")

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

func IsErrorResponse(statusCode int) bool {
	return statusCode >= 400
}
