package rsshub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	"golang.org/x/net/html/charset"
)

var (
	errInvalidQueryParameter   = errors.New("invalid query parameter")
	errInvalidMethod           = errors.New("invalid method")
	errInvalidHeaders          = errors.New("invalid headers")
	errInvalidFormData         = errors.New("invalid form data")
	errUnsupportedResponseType = errors.New("unsupported response type")
	errUnknownResponseType     = errors.New("unknown response type")
)

type response struct {
	URL     string `json:"url"`
	Body    any    `json:"body"`
	Data    any    `json:"data"`
	Data2   any    `json:"_data"`
	Headers any    `json:"headers"`
}

func (r *RSSHub) fetch(opts map[string]any) (*response, error) {
	rawUrl := toString(opts["url"])
	req := r.client.NewRequest()

	if query, ok := opts["query"]; ok {
		switch query := query.(type) {
		case map[string]any:
			for param, value := range query {
				req.SetQueryParam(param, toString(value))
			}
		case string:
			req.SetQueryString(query)
		case nil:
		default:
			return nil, errInvalidQueryParameter
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
			return nil, fmt.Errorf("%w %s", errInvalidMethod, m)
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
		headers, ok := headers.(map[string]any)
		if !ok {
			return nil, errInvalidHeaders
		}
		for header, value := range headers {
			req.SetHeader(header, toString(value))
		}
	}

	if form, ok := opts["form"]; ok {
		form, ok := form.(map[string]any)
		if !ok {
			return nil, errInvalidFormData
		}
		for key, value := range form {
			req.FormData.Set(key, toString(value))
		}
	}

	resp, err := req.Execute(method, rawUrl)
	if err != nil {
		return nil, err
	}
	body := resp.Body()

	response := new(response)
	switch toString(opts["responseType"]) {
	case "blob", "stream":
		return nil, errUnsupportedResponseType
	case "buffer":
		response.Body = body
		response.Data = body
	case "text":
		if body, err = tryDecode(body, opts); err != nil {
			return nil, err
		}
		response.Body = string(body)
		response.Data = response.Body
	case "json":
		if len(body) == 0 {
			response.Body = ""
			response.Data = ""
		} else {
			if body, err = tryDecode(body, opts); err != nil {
				return nil, err
			}
			if err = json.Unmarshal(body, &response.Body); err != nil {
				return nil, err
			}
			response.Data = response.Body
		}
	case "":
		if body, err = tryDecode(body, opts); err != nil {
			return nil, err
		}
		response.Body = string(body)
		if json.Unmarshal(body, &response.Data) != nil {
			response.Data = response.Body
		}
	default:
		return nil, errUnknownResponseType
	}

	response.URL = resp.Request.URL
	response.Data2 = response.Data
	response.Headers = resp.Header()
	return response, nil
}

func tryDecode(body []byte, opts map[string]any) ([]byte, error) {
	if e, _ := charset.Lookup(toString(opts["encoding"])); e != nil {
		return e.NewDecoder().Bytes(body)
	}
	return body, nil
}

func toString(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case []uint8:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
