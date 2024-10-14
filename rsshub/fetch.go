package rsshub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"rsslab/utils"
	"strings"

	"github.com/dop251/goja"
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

func (r *RSSHub) fetch(request, options goja.Value, method string, respFmt respFmt, vm *goja.Runtime) (any, error) {
	if request.ExportType().Kind() == reflect.String || vm.InstanceOf(request, vm.Get("URL").ToObject(vm)) {
		if options == nil || !options.ToBoolean() {
			options = vm.NewObject()
		}
		options.ToObject(vm).Set("url", request.String())
	} else {
		options = request
	}
	opts := options.ToObject(vm)

	rawUrl := opts.Get("url").String()
	baseUrl := opts.Get("baseURL")
	if baseUrl == nil {
		baseUrl = opts.Get("prefixUrl")
	}
	if baseUrl != nil {
		base, err := url.Parse(baseUrl.String())
		if err != nil {
			return nil, err
		}
		ref, err := url.Parse(rawUrl)
		if err != nil {
			return nil, err
		}
		rawUrl = base.ResolveReference(ref).String()
	}
	if method == "" {
		if m := opts.Get("method"); m != nil {
			m := strings.ToUpper(m.String())
			switch m {
			case "":
				method = http.MethodGet
			case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
				http.MethodDelete, http.MethodOptions, http.MethodPatch:
				method = m
			default:
				return nil, fmt.Errorf("%w %s", errInvalidMethod, m)
			}
		}
	}
	req, err := http.NewRequest(method, rawUrl, nil)
	if err != nil {
		return nil, err
	}

	query := opts.Get("query")
	if query == nil {
		query = opts.Get("searchParams")
	}
	if query != nil {
		if query.ExportType().Kind() == reflect.String {
			req.URL.RawQuery = query.String()
		} else {
			values := make(url.Values)
			query := query.ToObject(vm)
			var keys []string
			if err := vm.Try(func() { keys = query.Keys() }); err != nil {
				return nil, errInvalidQueryParameter
			}
			for _, key := range keys {
				values.Set(key, query.Get(key).String())
			}
			req.URL.RawQuery = values.Encode()
		}
	}

	req.Header.Set("User-Agent", utils.USER_AGENT)
	req.Header.Set("Referer", req.URL.Scheme+"://"+req.URL.Host)
	if headers := opts.Get("headers"); headers != nil {
		headers := headers.ToObject(vm)
		var keys []string
		if err := vm.Try(func() { keys = headers.Keys() }); err != nil {
			return nil, errInvalidHeaders
		}
		for _, key := range keys {
			req.Header.Set(key, headers.Get(key).String())
		}
	}

	var reqBody any
	if form := opts.Get("form"); form != nil {
		form := form.ToObject(vm)
		var keys []string
		if err := vm.Try(func() { keys = form.Keys() }); err != nil {
			return nil, errInvalidFormData
		}
		values := make(url.Values)
		for _, key := range keys {
			values.Set(key, form.Get(key).String())
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqBody = values.Encode()
	}
	if json := opts.Get("json"); json != nil {
		req.Header.Set("Content-Type", "application/json")
		reqBody = json.Export()
	}
	if body := opts.Get("body"); body != nil {
		reqBody = body.Export()
	}

	resp, body, err := r.do(req, reqBody)
	if err != nil {
		return nil, err
	}

	response := new(response)
	responseType := opts.Get("responseType")
	var respType string
	if responseType != nil {
		respType = responseType.String()
	}
	switch respType {
	case "blob", "stream", "buffer", "arrayBuffer":
		return nil, errUnsupportedResponseType
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
