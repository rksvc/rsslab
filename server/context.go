package server

import (
	"encoding"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
	"rsslab/utils"
	"strconv"
)

type context struct {
	w http.ResponseWriter
	r *http.Request
}

func (c *context) VarInt(name string) (int, error) {
	n, err := strconv.Atoi(c.r.PathValue(name))
	if err != nil {
		return 0, &errBadRequest{err}
	}
	return n, nil
}

func (c *context) ParseBody(v any) error {
	err := json.NewDecoder(c.r.Body).Decode(v)
	if err != nil {
		return &errBadRequest{err}
	}
	return nil
}

func (c *context) ParseQuery(v any) error {
	q := c.r.URL.Query()
	val := reflect.ValueOf(v).Elem()
	typ := val.Type()
	for i := range val.NumField() {
		if f := val.Field(i); f.CanSet() {
			if k := f.Kind(); k == reflect.Struct {
				if err := c.ParseQuery(f.Addr().Interface()); err != nil {
					return err
				}
			} else if key, ok := typ.Field(i).Tag.Lookup("query"); ok {
				if v := q.Get(key); v != "" {
					if k == reflect.Pointer && f.IsZero() {
						f.Set(reflect.New(f.Type().Elem()))
					}
					if f.CanConvert(reflect.TypeFor[encoding.TextUnmarshaler]()) {
						err := f.
							Interface().(encoding.TextUnmarshaler).
							UnmarshalText(utils.StringToBytes(v))
						if err != nil {
							return &errBadRequest{err}
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
								return &errBadRequest{errors.New("invalid bool value")}
							}
						case reflect.Int:
							n, err := strconv.Atoi(v)
							if err != nil {
								return &errBadRequest{err}
							}
							f.SetInt(int64(n))
						case reflect.String:
							f.SetString(v)
						default:
							panic("unsupported type " + k.String())
						}
					}
				}
			}
		}
	}
	return nil
}

func (c *context) Write(bytes []byte) error {
	_, err := c.w.Write(bytes)
	if err != nil {
		log.Print(err)
	}
	return nil
}

func (c *context) JSON(v any) error {
	c.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(c.w).Encode(v)
}

func (c *context) NotFound() error {
	c.w.WriteHeader(http.StatusNotFound)
	return nil
}
