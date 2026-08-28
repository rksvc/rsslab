package utils

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	lua "github.com/yuin/gopher-lua"
	"golang.org/x/net/html"
)

const USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"

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
	return fmt.Errorf(`%s %#v: %s`, resp.Request.Method, resp.Request.URL.String(), resp.Status)
}

func IsErrorResponse(statusCode int) bool {
	return statusCode >= 400
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

func UnmarshalLTable(t *lua.LTable, v any) error {
	val := reflect.ValueOf(v).Elem()
	typ := val.Type()
	for i := range val.NumField() {
		if f := val.Field(i); f.CanSet() {
			if key, ok := typ.Field(i).Tag.Lookup("json"); ok && key != "-" {
				key, _, _ = strings.Cut(key, ",")
				v := t.RawGetString(key)
				if v == lua.LNil {
					continue
				}
				unmarshalErr := func() error {
					return fmt.Errorf("cannot unmarshal %s into Go struct field .%s of type %T", v.Type(), key, f.Interface())
				}
				switch f.Kind() {
				case reflect.String:
					if s, ok := v.(lua.LString); ok {
						f.Set(reflect.ValueOf(s.String()))
					} else {
						return unmarshalErr()
					}
				case reflect.Slice:
					if t, ok := v.(*lua.LTable); ok {
						for i := range t.MaxN() {
							v := t.RawGetInt(i + 1)
							elem := reflect.New(f.Type().Elem())
							if t, ok := v.(*lua.LTable); ok {
								if err := UnmarshalLTable(t, elem.Interface()); err != nil {
									return err
								}
								f.Set(reflect.Append(f, elem.Elem()))
							} else {
								return fmt.Errorf("cannot unmarshal %s into .%s.%d of type %T", v.Type(), key, i, elem.Elem().Interface())
							}
						}
					} else {
						return unmarshalErr()
					}
				default:
					if _, ok := f.Interface().(*time.Time); ok {
						if s, ok := v.(lua.LString); ok {
							f.Set(reflect.ValueOf(ParseDate(s.String())))
						} else {
							return unmarshalErr()
						}
					} else {
						panic(fmt.Errorf("unsupported type %T", f.Interface()))
					}
				}
			}
		}
	}
	return nil
}
