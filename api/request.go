package api

import (
	"net/http"

	"github.com/HunanTV/eru-agent/logs"
)

type Request struct {
	http.Request
}

func (r *Request) Init() {
	r.ParseForm()
}

func NewRequest(r *http.Request) *Request {
	req := &Request{*r}
	req.Init()
	logs.Debug(req.Method, req.URL.Path)
	return req
}
