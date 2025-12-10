package utils

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
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

func ParseQuery(url *url.URL, v any) error {
	q := url.Query()
	val := reflect.ValueOf(v).Elem()
	typ := val.Type()
	for i := range val.NumField() {
		if f := val.Field(i); f.CanSet() {
			if k := f.Kind(); k == reflect.Struct {
				if err := ParseQuery(url, f.Addr().Interface()); err != nil {
					return err
				}
			} else if key, ok := typ.Field(i).Tag.Lookup("json"); ok {
				if v := q.Get(key); v != "" {
					if k == reflect.Pointer && f.IsZero() {
						f.Set(reflect.New(f.Type().Elem()))
					}
					if f.CanConvert(reflect.TypeFor[encoding.TextUnmarshaler]()) {
						err := f.
							Interface().(encoding.TextUnmarshaler).
							UnmarshalText(StringToBytes(v))
						if err != nil {
							return err
						}
					} else {
						if k == reflect.Pointer {
							f = f.Elem()
							k = f.Kind()
						}
						switch k {
						case reflect.Bool:
							switch v {
							case "true":
								f.SetBool(true)
							case "false":
								f.SetBool(false)
							default:
								return errors.New("invalid bool value")
							}
						case reflect.Int:
							n, err := strconv.Atoi(v)
							if err != nil {
								return err
							}
							f.SetInt(int64(n))
						case reflect.String:
							f.SetString(v)
						case reflect.Map:
							val, ok := f.Addr().Interface().(*map[string]string)
							if !ok {
								panic(fmt.Errorf("unsupported type %T", f.Interface()))
							}
							if err := json.Unmarshal(StringToBytes(v), val); err != nil {
								return err
							}
						default:
							panic(fmt.Errorf("unsupported type %T", f.Interface()))
						}
					}
				}
			}
		}
	}
	return nil
}
