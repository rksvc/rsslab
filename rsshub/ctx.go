package rsshub

import "github.com/dop251/goja"

type Ctx struct {
	Req Req `json:"req"`
}

func NewCtx(path string, params, queries map[string]string) *Ctx {
	return &Ctx{Req: Req{
		Path: path,

		params:  params,
		queries: queries,
	}}
}

func (ctx *Ctx) Set() {}

type Req struct {
	params  map[string]string
	queries map[string]string

	Path string `json:"path"`
}

func (req *Req) Param(key *string) any {
	if key == nil {
		return req.params
	} else if param, ok := req.params[*key]; ok {
		return param
	}
	return goja.Undefined()
}

func (req *Req) Query(key *string) any {
	if key == nil {
		return req.queries
	} else if query, ok := req.queries[*key]; ok {
		return query
	}
	return goja.Undefined()
}
