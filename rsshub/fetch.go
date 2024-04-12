package rsshub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	"golang.org/x/net/html/charset"
)

var ErrInvalidQueryParameter = errors.New("invalid query parameter")
var ErrInvalidMethod = errors.New("invalid method")
var ErrInvalidHeaders = errors.New("invalid headers")
var ErrInvalidFormData = errors.New("invalid form data")
var ErrUnsupportedResponseType = errors.New("unsupported response type")
var ErrUnknownResponseType = errors.New("unknown response type")

type ErrorResponse struct {
	Method, URL string
	Status      int
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf(`%s "%s": %s`, r.Method, r.URL, http.StatusText(r.Status))
}

type response struct {
	URL     string  `json:"url"`
	Body    any     `json:"body"`
	Data    any     `json:"data"`
	Data2   any     `json:"_data"`
	Headers headers `json:"headers"`
}

type headers struct {
	http.Header
}

func (h headers) GetSetCookie() []string {
	return h.Values("Set-Cookie")
}

func (r *RSSHub) fetch(opts map[string]any) (*response, error) {
	rawUrl := toString(opts["url"])
	req := r.R()

	if queryParams, ok := opts["query"]; ok {
		switch queryParams := queryParams.(type) {
		case map[string]any:
			for param, value := range queryParams {
				req.SetQueryParam(param, toString(value))
			}
		case string:
			req.SetQueryString(queryParams)
		case nil:
		default:
			return nil, ErrInvalidQueryParameter
		}
	}

	if baseUrl, ok := opts["baseURL"]; ok {
		base, err := url.Parse(toString(baseUrl))
		if err != nil {
			return nil, err
		}
		ref, err := url.Parse(rawUrl)
		if err != nil {
			return nil, err
		}
		rawUrl = base.ResolveReference(ref).String()
	}

	method := resty.MethodGet
	if m, ok := opts["method"]; ok {
		m := strings.ToUpper(toString(m))
		switch m {
		case "":
		case resty.MethodGet, resty.MethodHead, resty.MethodPost, resty.MethodPut,
			resty.MethodDelete, resty.MethodOptions, resty.MethodPatch:
			method = m
		default:
			return nil, fmt.Errorf("%w %s", ErrInvalidMethod, m)
		}
	}

	if url, err := url.Parse(rawUrl); err == nil {
		url.Path = ""
		url.RawQuery = ""
		url.Fragment = ""
		req.SetHeader("Referer", url.String())
	}

	if body, ok := opts["body"]; ok {
		req.SetBody(body)
	}

	if json, ok := opts["json"]; ok {
		req.SetHeader("Content-Type", "application/json")
		req.SetBody(json)
	}

	if headers, ok := opts["headers"]; ok {
		if headers, ok := headers.(map[string]any); ok {
			for header, value := range headers {
				req.SetHeader(header, toString(value))
			}
		} else {
			return nil, ErrInvalidHeaders
		}
	}

	if form, ok := opts["form"]; ok {
		if form, ok := form.(map[string]any); ok {
			for key, value := range form {
				req.FormData.Set(key, toString(value))
			}
		} else {
			return nil, ErrInvalidFormData
		}
	}

	resp, err := req.Execute(method, rawUrl)
	if err != nil {
		return nil, err
	} else if status := resp.StatusCode(); status < 200 || status >= 300 {
		return nil, &ErrorResponse{method, rawUrl, status}
	}
	body := resp.Body()

	response := new(response)
	switch toString(opts["responseType"]) {
	case "blob", "stream":
		return nil, ErrUnsupportedResponseType
	case "buffer":
		response.Body = body
		response.Data = body
	case "text":
		if err = decode(&body, opts); err != nil {
			return nil, err
		}
		response.Body = string(body)
		response.Data = response.Body
	case "json":
		if len(body) == 0 {
			response.Body = ""
			response.Data = ""
		} else {
			if err = decode(&body, opts); err != nil {
				return nil, err
			} else if err = json.Unmarshal(body, &response.Body); err != nil {
				return nil, err
			}
			response.Data = response.Body
		}
	case "":
		if err = decode(&body, opts); err != nil {
			return nil, err
		}
		response.Body = string(body)
		var data any
		if json.Unmarshal(body, &data) == nil {
			response.Data = data
		} else {
			response.Data = response.Body
		}
	default:
		return nil, ErrUnknownResponseType
	}

	response.URL = resp.Request.URL
	response.Data2 = response.Data
	response.Headers = headers{resp.Header()}
	return response, nil
}

func decode(body *[]byte, opts map[string]any) error {
	if e, _ := charset.Lookup(toString(opts["encoding"])); e != nil {
		b, err := e.NewDecoder().Bytes(*body)
		if err != nil {
			return err
		}
		*body = b
	}
	return nil
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
