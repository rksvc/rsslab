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
	errUnsupportedResponseType = errors.New("unsupported response type")
	errUnknownResponseType     = errors.New("unknown response type")
)

type options struct {
	Url          string            `json:"url"`
	BaseURL      string            `json:"baseURL"`
	PrefixUrl    string            `json:"prefixUrl"`
	Method       string            `json:"method"`
	Query        any               `json:"query"`
	SearchParams any               `json:"searchParams"`
	Headers      map[string]string `json:"headers"`
	Form         map[string]string `json:"form"`
	Json         any               `json:"json"`
	Body         any               `json:"body"`
	ResponseType string            `json:"responseType"`
}

type response struct {
	URL     string         `json:"url,omitempty"`
	Body    any            `json:"body,omitempty"`
	Data    any            `json:"data,omitempty"`
	Data2   any            `json:"_data,omitempty"`
	Headers map[string]any `json:"headers,omitempty"`
}

type respFmt int

const (
	respFmtOfetch respFmt = iota
	respFmtOfetchRaw
	respFmtGot
)

func (r *RSSHub) fetch(opts *options, respFmt respFmt) (any, error) {
	if opts.BaseURL == "" {
		opts.BaseURL = opts.PrefixUrl
	}
	if opts.BaseURL != "" {
		base, err := url.Parse(opts.BaseURL)
		if err != nil {
			return nil, err
		}
		ref, err := url.Parse(opts.Url)
		if err != nil {
			return nil, err
		}
		opts.Url = base.ResolveReference(ref).String()
	}
	req, err := http.NewRequest(strings.ToUpper(opts.Method), opts.Url, nil)
	if err != nil {
		return nil, err
	}

	if opts.Query == nil {
		opts.Query = opts.SearchParams
	}
	if opts.Query != nil {
		switch query := opts.Query.(type) {
		case string:
			req.URL.RawQuery = query
		case map[string]any:
			values := make(url.Values)
			for param, val := range query {
				values.Set(param, fmt.Sprintf("%v", val))
			}
			req.URL.RawQuery = values.Encode()
		default:
			return nil, errInvalidQueryParameter
		}
	}

	req.Header.Set("User-Agent", utils.USER_AGENT)
	req.Header.Set("Referer", req.URL.Scheme+"://"+req.URL.Host)
	for key, val := range opts.Headers {
		req.Header.Set(key, val)
	}

	var reqBody any
	if opts.Form != nil {
		values := make(url.Values)
		for key, val := range opts.Form {
			values.Set(key, val)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqBody = values.Encode()
	}
	if opts.Json != nil {
		req.Header.Set("Content-Type", "application/json")
		reqBody = opts.Json
	}
	if opts.Body != nil {
		reqBody = opts.Body
	}

	resp, body, err := r.do(req, reqBody)
	if err != nil {
		return nil, err
	}

	response := new(response)
	switch opts.ResponseType {
	case "blob", "stream":
		return nil, errUnsupportedResponseType
	case "buffer", "arrayBuffer":
		switch respFmt {
		case respFmtOfetch:
			return body, nil
		case respFmtOfetchRaw:
			response.Data2 = body
		case respFmtGot:
			response.Body = body
			response.Data = body
		}
	case "text":
		switch respFmt {
		case respFmtOfetch:
			return utils.BytesToString(body), nil
		case respFmtOfetchRaw:
			response.Data2 = utils.BytesToString(body)
		case respFmtGot:
			response.Body = utils.BytesToString(body)
			if len(body) == 0 || json.Unmarshal(body, &response.Data) != nil {
				response.Data = response.Body
			}
		}
	case "json", "":
		var data any
		if len(body) == 0 || json.Unmarshal(body, &data) != nil {
			data = utils.BytesToString(body)
		}
		switch respFmt {
		case respFmtOfetch:
			return data, nil
		case respFmtOfetchRaw:
			response.Data2 = data
		case respFmtGot:
			response.Body = utils.BytesToString(body)
			response.Data = data
		}
	default:
		return nil, errUnknownResponseType
	}

	if respFmt == respFmtOfetchRaw {
		response.URL = req.URL.String()
		response.Headers = map[string]any{
			"getSetCookie": func() []string {
				return resp.Header.Values("Set-Cookie")
			},
			"get": func(key string) string {
				return resp.Header.Get(key)
			},
		}
	}
	return response, nil
}
