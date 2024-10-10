package rsshub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"rsslab/utils"
	"strings"
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
	method := http.MethodGet
	if m, ok := opts["method"]; ok {
		m := strings.ToUpper(toString(m))
		switch m {
		case "":
		case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
			http.MethodDelete, http.MethodOptions, http.MethodPatch:
			method = m
		default:
			return nil, fmt.Errorf("%w %s", errInvalidMethod, m)
		}
	}
	req, err := http.NewRequest(method, rawUrl, nil)
	if err != nil {
		return nil, err
	}

	if query, ok := opts["query"]; ok {
		switch query := query.(type) {
		case map[string]any:
			values := url.Values{}
			for param, value := range query {
				values.Set(param, toString(value))
			}
			req.URL.RawQuery = values.Encode()
		case string:
			req.URL.RawQuery = query
		case nil:
		default:
			return nil, errInvalidQueryParameter
		}
	}

	req.Header.Set("User-Agent", utils.UserAgent)
	req.Header.Set("Referer", req.URL.Scheme+"://"+req.URL.Host)
	if headers, ok := opts["headers"]; ok {
		headers, ok := headers.(map[string]any)
		if !ok {
			return nil, errInvalidHeaders
		}
		for header, value := range headers {
			req.Header.Set(header, toString(value))
		}
	}

	var reqBody any
	if form, ok := opts["form"]; ok {
		form, ok := form.(map[string]any)
		if !ok {
			return nil, errInvalidFormData
		}
		values := url.Values{}
		for key, value := range form {
			values.Set(key, toString(value))
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqBody = values.Encode()
	}
	if json, ok := opts["json"]; ok {
		req.Header.Set("Content-Type", "application/json")
		reqBody = json
	}
	if body, ok := opts["body"]; ok {
		reqBody = body
	}

	resp, body, err := r.do(req, reqBody)
	if err != nil {
		return nil, err
	}

	response := new(response)
	switch toString(opts["responseType"]) {
	case "blob", "stream":
		return nil, errUnsupportedResponseType
	case "buffer":
		response.Body = body
		response.Data = body
	case "text":
		response.Body = string(body)
		response.Data = response.Body
	case "json":
		if len(body) == 0 {
			response.Body = ""
			response.Data = ""
		} else {
			if err = json.Unmarshal(body, &response.Body); err != nil {
				return nil, err
			}
			response.Data = response.Body
		}
	case "":
		response.Body = string(body)
		if json.Unmarshal(body, &response.Data) != nil {
			response.Data = response.Body
		}
	default:
		return nil, errUnknownResponseType
	}

	response.URL = req.URL.String()
	response.Data2 = response.Data
	response.Headers = resp.Header
	return response, nil
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
