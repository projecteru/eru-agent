package api

import (
	"net/http"

	"github.com/HunanTV/eru-agent/logs"
)

type Request struct {
	http.Request
	Start int
	Limit int
}

func (r *Request) Init() {
	r.Start = Atoi(r.Form.Get("start"), 0)
	r.Limit = Atoi(r.Form.Get("limit"), 20)
}

func NewRequest(r *http.Request) *Request {
	req := &Request{*r, 0, 20}
	req.Init()
	logs.Debug(req.Method, req.URL.Path)
	return req
}
